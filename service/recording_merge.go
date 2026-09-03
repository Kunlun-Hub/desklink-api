package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

const recordingMergeDelay = 5 * time.Second

var recordingMergeTimers = struct {
	sync.Mutex
	values map[string]*time.Timer
}{values: make(map[string]*time.Timer)}

var recordingMergeRun sync.Mutex

type recordingSessionKey struct {
	PeerId   string
	FromPeer string
	Session  string `gorm:"column:session_id"`
}

func recordingMergeKey(recording *model.SessionRecording) string {
	if recording == nil || recording.SessionId == "" || recording.PeerId == "" || recording.FromPeer == "" {
		return ""
	}
	return recording.PeerId + "\x00" + recording.FromPeer + "\x00" + recording.SessionId
}

func (s *RecordingService) scheduleSessionMerge(recording *model.SessionRecording) {
	key := recordingMergeKey(recording)
	if key == "" {
		return
	}
	recordingMergeTimers.Lock()
	if timer := recordingMergeTimers.values[key]; timer != nil {
		timer.Stop()
	}
	recordingMergeTimers.values[key] = time.AfterFunc(recordingMergeDelay, func() {
		recordingMergeTimers.Lock()
		delete(recordingMergeTimers.values, key)
		recordingMergeTimers.Unlock()
		if err := s.MergeSession(recording); err != nil {
			Logger.Warnf("recording session merge failed for %s: %v", key, err)
		}
	})
	recordingMergeTimers.Unlock()
}

func (s *RecordingService) MergeSession(recording *model.SessionRecording) error {
	recordingMergeRun.Lock()
	defer recordingMergeRun.Unlock()
	key := recordingMergeKey(recording)
	if key == "" {
		return nil
	}
	var activeSegments int64
	if err := DB.Model(&model.SessionRecording{}).
		Where("peer_id = ? AND from_peer = ? AND session_id = ? AND status IN ?",
			recording.PeerId, recording.FromPeer, recording.SessionId,
			[]string{model.RecordingStatusUploading, model.RecordingStatusTranscoding}).
		Count(&activeSegments).Error; err != nil {
		return err
	}
	if activeSegments > 0 {
		return nil
	}
	var segments []*model.SessionRecording
	if err := DB.Where("peer_id = ? AND from_peer = ? AND session_id = ? AND status = ?",
		recording.PeerId, recording.FromPeer, recording.SessionId, model.RecordingStatusComplete).
		Order("started_at asc, id asc").Find(&segments).Error; err != nil {
		return err
	}
	if len(segments) < 2 {
		return nil
	}
	primary := segments[0]
	mergedCursorTrack := make([]RecordingCursorSample, 0)
	trackOffset := int64(0)
	for _, segment := range segments {
		if segment.CursorTrack != "" {
			var samples []RecordingCursorSample
			if err := json.Unmarshal([]byte(segment.CursorTrack), &samples); err != nil {
				return fmt.Errorf("decode recording cursor track: %w", err)
			}
			for _, sample := range samples {
				sample.Time += trackOffset
				mergedCursorTrack = append(mergedCursorTrack, sample)
			}
		}
		trackOffset += segment.DurationMs
	}
	cursorTrackJSON, err := json.Marshal(mergedCursorTrack)
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "desklink-recording-merge-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	cleanups := make([]func(), 0, len(segments))
	segmentPaths := make([]string, 0, len(segments))
	for _, segment := range segments {
		value, cleanup, materializeErr := s.MaterializeRecordingObject(segment, false)
		if materializeErr != nil {
			for _, clean := range cleanups {
				clean()
			}
			return materializeErr
		}
		cleanups = append(cleanups, cleanup)
		value, err = filepath.Abs(value)
		if err != nil {
			for _, clean := range cleanups {
				clean()
			}
			return err
		}
		segmentPaths = append(segmentPaths, value)
	}
	for _, clean := range cleanups {
		defer clean()
	}

	mergedPath := filepath.Join(tempDir, primary.UploadId+".merged.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	if err = s.reencodeRecordingSegments(ctx, segmentPaths, tempDir, mergedPath); err != nil {
		return err
	}
	codec := "h264"
	container := "mp4"

	mergedName := primary.UploadId + ".merged." + container
	if err = s.archiveRecordingObject(primary, mergedName, mergedPath); err != nil {
		return fmt.Errorf("archive merged recording: %w", err)
	}
	mergedInfo, err := os.Open(mergedPath)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, mergedInfo)
	closeErr := mergedInfo.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	stat, err := os.Stat(mergedPath)
	if err != nil {
		return err
	}
	duration, err := s.probeRecordingDuration(ctx, mergedPath)
	if err != nil {
		return err
	}
	// Switch metadata first. If cleanup fails, the merged object remains the
	// canonical playable copy and the old objects can be retried later.
	if err = DB.Model(primary).Updates(map[string]interface{}{
		"storage_name": mergedName, "preview_storage_name": "", "container": container,
		"codec": codec, "size": stat.Size(), "duration_ms": duration,
		"sha256": hex.EncodeToString(hash.Sum(nil)), "status": model.RecordingStatusComplete,
		"cursor_track": string(cursorTrackJSON),
	}).Error; err != nil {
		return err
	}
	var cleanupErr error
	for _, segment := range segments {
		if segment.StorageName != mergedName {
			if err = s.deleteRecordingObject(segment, segment.StorageName); err != nil && cleanupErr == nil {
				cleanupErr = err
			}
		}
		if segment.PreviewStorageName != "" && segment.PreviewStorageName != mergedName {
			if err = s.deleteRecordingObject(segment, segment.PreviewStorageName); err != nil && cleanupErr == nil {
				cleanupErr = err
			}
		}
	}
	if err = DB.Where("id IN ?", func() []uint {
		ids := make([]uint, 0, len(segments)-1)
		for _, segment := range segments[1:] {
			ids = append(ids, segment.Id)
		}
		return ids
	}()).Delete(&model.SessionRecording{}).Error; err != nil {
		return err
	}
	if cleanupErr != nil {
		return fmt.Errorf("merged recording cleanup: %w", cleanupErr)
	}
	return nil
}

func (s *RecordingService) reencodeRecordingSegments(ctx context.Context, segmentPaths []string, tempDir, mergedPath string) error {
	if len(segmentPaths) == 0 {
		return errors.New("recording segments are required")
	}
	dimensions, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", segmentPaths[0]).Output()
	if err != nil {
		return fmt.Errorf("probe segment dimensions: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(dimensions)), "x")
	if len(parts) != 2 {
		return errors.New("invalid segment dimensions")
	}
	width, widthErr := strconv.Atoi(parts[0])
	height, heightErr := strconv.Atoi(parts[1])
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return errors.New("invalid segment dimensions")
	}
	width -= width % 2
	height -= height % 2
	listPath := filepath.Join(tempDir, "normalized.txt")
	listFile, err := os.OpenFile(listPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	for index, segmentPath := range segmentPaths {
		normalizedPath := filepath.Join(tempDir, fmt.Sprintf("normalized-%04d.ts", index))
		filter := fmt.Sprintf("fps=30,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2,setsar=1,setpts=N/(30*TB)", width, height, width, height)
		args := []string{"-hide_banner", "-loglevel", "error", "-y", "-i", segmentPath, "-map", "0:v:0", "-an", "-vf", filter, "-c:v", "libx264", "-preset", "veryfast", "-crf", "28", "-fps_mode", "cfr", "-f", "mpegts", normalizedPath}
		if output, runErr := exec.CommandContext(ctx, s.ffmpegPath(), args...).CombinedOutput(); runErr != nil {
			_ = listFile.Close()
			return fmt.Errorf("normalize segment %d: %v (%s)", index, runErr, strings.TrimSpace(string(output)))
		}
		if _, err = fmt.Fprintf(listFile, "file '%s'\n", strings.ReplaceAll(normalizedPath, "'", "'\\''")); err != nil {
			_ = listFile.Close()
			return err
		}
	}
	if err = listFile.Close(); err != nil {
		return err
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", "-movflags", "+faststart", mergedPath}
	if output, runErr := exec.CommandContext(ctx, s.ffmpegPath(), args...).CombinedOutput(); runErr != nil {
		return fmt.Errorf("concat normalized segments: %v (%s)", runErr, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *RecordingService) probeRecordingDuration(ctx context.Context, recordingPath string) (int64, error) {
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nw=1:nk=1", recordingPath).Output()
	if err != nil {
		return 0, fmt.Errorf("probe merged recording duration: %w", err)
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || value <= 0 {
		return 0, errors.New("invalid merged recording duration")
	}
	return int64(value * 1000), nil
}

// MergeExistingSessions repairs segments created before session merging was
// enabled. It runs once in the background after startup; future segments use
// scheduleSessionMerge instead.
func (s *RecordingService) mergeableSessionKeys() ([]recordingSessionKey, error) {
	var keys []recordingSessionKey
	err := DB.Model(&model.SessionRecording{}).
		Select("peer_id, from_peer, session_id").
		Where("status = ? AND session_id <> '' AND from_peer <> ''", model.RecordingStatusComplete).
		Group("peer_id, from_peer, session_id").
		Having("COUNT(*) > 1").Find(&keys).Error
	return keys, err
}

func (s *RecordingService) MergeExistingSessions() {
	keys, err := s.mergeableSessionKeys()
	if err != nil {
		Logger.Warnf("recording session merge scan failed: %v", err)
		return
	}
	for _, key := range keys {
		var recording model.SessionRecording
		if err := DB.Where("peer_id = ? AND from_peer = ? AND session_id = ?", key.PeerId, key.FromPeer, key.Session).Order("started_at asc, id asc").First(&recording).Error; err != nil {
			Logger.Warnf("recording session merge lookup failed: %v", err)
			continue
		}
		if err := s.MergeSession(&recording); err != nil {
			Logger.Warnf("recording historical session merge failed for %s: %v", key.Session, err)
		}
	}
}
