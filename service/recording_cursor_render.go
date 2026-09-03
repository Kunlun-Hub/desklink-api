package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

const (
	recordingCursorGenerating = "generating"
	recordingCursorReady      = "ready"
	recordingCursorFailed     = "failed"
)

func (s *RecordingService) StartCursorRender(recording *model.SessionRecording, retry bool) (string, error) {
	if recording.Status != model.RecordingStatusComplete {
		return "", errors.New("recording is not complete")
	}
	var samples []RecordingCursorSample
	if err := json.Unmarshal([]byte(recording.CursorTrack), &samples); err != nil || len(samples) == 0 {
		return "", errors.New("this recording has no cursor track")
	}
	if recording.CursorStorageName != "" && recording.CursorRenderStatus == recordingCursorReady {
		return recordingCursorReady, nil
	}
	if recording.CursorRenderStatus == recordingCursorGenerating {
		return recordingCursorGenerating, nil
	}
	if recording.CursorRenderStatus == recordingCursorFailed && !retry {
		return recordingCursorFailed, nil
	}
	result := DB.Model(&model.SessionRecording{}).
		Where("id = ? AND cursor_render_status <> ?", recording.Id, recordingCursorGenerating).
		Updates(map[string]interface{}{"cursor_render_status": recordingCursorGenerating, "cursor_render_error": ""})
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected > 0 {
		go s.renderCursorRecording(recording.Id)
	}
	return recordingCursorGenerating, nil
}

func (s *RecordingService) renderCursorRecording(id uint) {
	defer func() {
		if recording, lookupErr := s.Info(id); lookupErr == nil {
			s.scheduleSessionMerge(recording)
		}
	}()
	recording, err := s.Info(id)
	if err == nil {
		err = s.renderCursorRecordingFile(recording)
	}
	if err == nil {
		return
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 500 {
		message = message[len(message)-500:]
	}
	_ = DB.Model(&model.SessionRecording{}).Where("id = ?", id).Updates(map[string]interface{}{
		"cursor_render_status": recordingCursorFailed, "cursor_render_error": message,
	}).Error
}

func (s *RecordingService) renderCursorRecordingFile(recording *model.SessionRecording) error {
	var samples []RecordingCursorSample
	if err := json.Unmarshal([]byte(recording.CursorTrack), &samples); err != nil || len(samples) == 0 {
		return errors.New("this recording has no cursor track")
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].Time < samples[j].Time })
	source, cleanup, err := s.MaterializeRecordingObject(recording, false)
	if err != nil {
		return fmt.Errorf("materialize recording: %w", err)
	}
	defer cleanup()
	source, err = filepath.Abs(source)
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "desklink-recording-cursor-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	width, height, err := probeRecordingDimensions(source)
	if err != nil {
		return err
	}
	duration := recording.DurationMs
	if duration <= 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		duration, err = s.probeRecordingDuration(ctx, source)
		cancel()
		if err != nil {
			return err
		}
	}
	assPath := filepath.Join(tempDir, "cursor.ass")
	if err = writeCursorASS(assPath, samples, duration, width, height); err != nil {
		return err
	}
	outputName := recording.UploadId + ".cursor.mp4"
	outputPath := filepath.Join(tempDir, outputName)
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.ffmpegPath(), "-hide_banner", "-loglevel", "error", "-y",
		"-i", source, "-vf", "ass=cursor.ass", "-map", "0:v:0", "-map", "0:a?",
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-movflags", "+faststart", outputPath)
	cmd.Dir = tempDir
	if output, runErr := cmd.CombinedOutput(); runErr != nil {
		return fmt.Errorf("render cursor recording: %v (%s)", runErr, strings.TrimSpace(string(output)))
	}
	if err = s.archiveRecordingObject(recording, outputName, outputPath); err != nil {
		return fmt.Errorf("archive cursor recording: %w", err)
	}
	if err = DB.Model(&model.SessionRecording{}).Where("id = ?", recording.Id).Updates(map[string]interface{}{
		"cursor_storage_name": outputName, "cursor_render_status": recordingCursorReady, "cursor_render_error": "",
	}).Error; err != nil {
		_ = s.deleteRecordingObject(recording, outputName)
		return err
	}
	s.removeStagedRecordingObject(recording, outputName)
	return nil
}

func probeRecordingDimensions(source string) (int, int, error) {
	output, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", source).Output()
	if err != nil {
		return 0, 0, fmt.Errorf("probe recording dimensions: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "x")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid recording dimensions")
	}
	width, widthErr := strconv.Atoi(parts[0])
	height, heightErr := strconv.Atoi(parts[1])
	if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
		return 0, 0, errors.New("invalid recording dimensions")
	}
	return width, height, nil
}

func writeCursorASS(path string, samples []RecordingCursorSample, durationMs int64, width, height int) error {
	var value strings.Builder
	fmt.Fprintf(&value, "[Script Info]\nScriptType: v4.00+\nPlayResX: %d\nPlayResY: %d\nScaledBorderAndShadow: yes\n\n", width, height)
	value.WriteString("[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\nStyle: Cursor,Arial,20,&H00000000,&H00000000,&H00FFFFFF,&H00000000,0,0,0,0,100,100,0,0,1,1.5,0,7,0,0,0,1\n\n")
	value.WriteString("[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	written := 0
	for index, sample := range samples {
		start := max(sample.Time, int64(0))
		end := durationMs
		if index+1 < len(samples) {
			end = samples[index+1].Time
		}
		if !sample.Visible || start >= durationMs || end <= start {
			continue
		}
		if end > durationMs {
			end = durationMs
		}
		x := int(uint64(sample.X) * uint64(width-1) / uint64(^uint16(0)))
		y := int(uint64(sample.Y) * uint64(height-1) / uint64(^uint16(0)))
		fmt.Fprintf(&value, "Dialogue: 0,%s,%s,Cursor,,0,0,0,,{\\an7\\pos(%d,%d)\\p1}m 0 0 l 0 24 6 18 11 29 16 27 11 16 20 16{\\p0}\n",
			assTimestamp(start), assTimestamp(end), x, y)
		written++
	}
	if written == 0 {
		return errors.New("this recording has no visible cursor samples")
	}
	return os.WriteFile(path, []byte(value.String()), 0600)
}

func assTimestamp(milliseconds int64) string {
	centiseconds := max(milliseconds, int64(0)) / 10
	hours := centiseconds / 360000
	minutes := centiseconds / 6000 % 60
	seconds := centiseconds / 100 % 60
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds%100)
}

func (s *RecordingService) MaterializeCursorRecording(recording *model.SessionRecording) (string, func(), error) {
	if recording.CursorRenderStatus != recordingCursorReady || recording.CursorStorageName == "" {
		return "", func() {}, errors.New("cursor recording is not ready")
	}
	return s.materializeRecordingNamedObject(recording, recording.CursorStorageName)
}
