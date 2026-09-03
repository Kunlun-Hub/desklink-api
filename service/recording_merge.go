package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	key := recordingMergeKey(recording)
	if key == "" {
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
	if primary.StorageSettingId != recording.StorageSettingId {
		return errors.New("recording segments use different storage settings")
	}

	tempDir, err := os.MkdirTemp("", "desklink-recording-merge-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	listPath := filepath.Join(tempDir, "concat.txt")
	listFile, err := os.OpenFile(listPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	cleanups := make([]func(), 0, len(segments))
	for _, segment := range segments {
		value, cleanup, materializeErr := s.MaterializeRecordingObject(segment, false)
		if materializeErr != nil {
			_ = listFile.Close()
			for _, clean := range cleanups {
				clean()
			}
			return materializeErr
		}
		cleanups = append(cleanups, cleanup)
		if _, err = fmt.Fprintf(listFile, "file '%s'\n", strings.ReplaceAll(value, "'", "'\\''")); err != nil {
			_ = listFile.Close()
			for _, clean := range cleanups {
				clean()
			}
			return err
		}
	}
	if err = listFile.Close(); err != nil {
		for _, clean := range cleanups {
			clean()
		}
		return err
	}
	for _, clean := range cleanups {
		defer clean()
	}

	mergedPath := filepath.Join(tempDir, primary.UploadId+".merged.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	copyArgs := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", listPath, "-c", "copy", "-movflags", "+faststart", mergedPath}
	codec := primary.Codec
	container := primary.Container
	if output, runErr := exec.CommandContext(ctx, s.ffmpegPath(), copyArgs...).CombinedOutput(); runErr != nil {
		fallbackPath := filepath.Join(tempDir, primary.UploadId+".merged.mp4")
		fallbackArgs := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "concat", "-safe", "0", "-i", listPath, "-c:v", "libx264", "-preset", "veryfast", "-crf", "28", "-an", "-movflags", "+faststart", fallbackPath}
		if fallbackOutput, fallbackErr := exec.CommandContext(ctx, s.ffmpegPath(), fallbackArgs...).CombinedOutput(); fallbackErr != nil {
			return fmt.Errorf("stream copy: %v (%s); re-encode: %v (%s)", runErr, strings.TrimSpace(string(output)), fallbackErr, strings.TrimSpace(string(fallbackOutput)))
		}
		mergedPath = fallbackPath
		codec = "h264"
		container = "mp4"
	}

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
	duration := int64(0)
	for _, segment := range segments {
		duration += segment.DurationMs
	}
	// Switch metadata first. If cleanup fails, the merged object remains the
	// canonical playable copy and the old objects can be retried later.
	if err = DB.Model(primary).Updates(map[string]interface{}{
		"storage_name": mergedName, "preview_storage_name": "", "container": container,
		"codec": codec, "size": stat.Size(), "duration_ms": duration,
		"sha256": hex.EncodeToString(hash.Sum(nil)), "status": model.RecordingStatusComplete,
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
