package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const defaultRecordingChunkSize int64 = 8 * 1024 * 1024
const minRecordingChunkSize int64 = 4 * 1024 * 1024
const maxRecordingChunkSize int64 = 8 * 1024 * 1024

type RecordingService struct{}

type RecordingInit struct {
	UploadId    string `json:"upload_id" binding:"omitempty,max=36"`
	UploadToken string `json:"upload_token" binding:"omitempty,max=128"`
	PeerId      string `json:"peer_id" binding:"required,max=64"`
	Uuid        string `json:"uuid" binding:"required,max=255"`
	FromPeer    string `json:"from_peer" binding:"max=64"`
	FromName    string `json:"from_name" binding:"max=255"`
	SessionId   string `json:"session_id" binding:"max=128"`
	Filename    string `json:"filename" binding:"required,max=255"`
	Codec       string `json:"codec" binding:"max=16"`
	StartedAt   int64  `json:"started_at"`
}

func (s *RecordingService) storagePath() string {
	path := strings.TrimSpace(Config.Recording.Path)
	if path == "" {
		path = "runtime/recordings"
	}
	return path
}

func (s *RecordingService) MaxChunkSize() int64 {
	if Config.Recording.MaxChunkSize > 0 {
		if Config.Recording.MaxChunkSize < minRecordingChunkSize {
			return minRecordingChunkSize
		}
		if Config.Recording.MaxChunkSize > maxRecordingChunkSize {
			return maxRecordingChunkSize
		}
		return Config.Recording.MaxChunkSize
	}
	return defaultRecordingChunkSize
}

func (s *RecordingService) Policy() (*model.RecordingPolicy, []string, error) {
	policy := &model.RecordingPolicy{}
	err := DB.First(policy, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy = &model.RecordingPolicy{Id: 1, Mode: model.RecordingModeOff, RetentionDays: 30}
		if err = DB.Create(policy).Error; err != nil {
			return nil, nil, err
		}
	} else if err != nil {
		return nil, nil, err
	}
	var peerIds []string
	if err = DB.Model(&model.RecordingPolicyDevice{}).Order("peer_id").Pluck("peer_id", &peerIds).Error; err != nil {
		return nil, nil, err
	}
	return policy, peerIds, nil
}

func (s *RecordingService) IsEnabled(peerId string) (bool, error) {
	policy, _, err := s.Policy()
	if err != nil {
		return false, err
	}
	switch policy.Mode {
	case model.RecordingModeAll:
		return true, nil
	case model.RecordingModeSelected:
		var count int64
		err = DB.Model(&model.RecordingPolicyDevice{}).Where("peer_id = ?", peerId).Count(&count).Error
		return count > 0, err
	default:
		return false, nil
	}
}

func (s *RecordingService) SavePolicy(mode string, retentionDays int, peerIds []string) error {
	if mode != model.RecordingModeOff && mode != model.RecordingModeAll && mode != model.RecordingModeSelected {
		return errors.New("invalid recording policy mode")
	}
	if retentionDays < 1 || retentionDays > 3650 {
		return errors.New("retention_days must be between 1 and 3650")
	}
	clean := make(map[string]struct{}, len(peerIds))
	for _, peerId := range peerIds {
		peerId = strings.TrimSpace(peerId)
		if peerId != "" && len(peerId) <= 64 {
			clean[peerId] = struct{}{}
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		policy := &model.RecordingPolicy{Id: 1, Mode: mode, RetentionDays: retentionDays}
		if err := tx.Save(policy).Error; err != nil {
			return err
		}
		if err := tx.Where("1 = 1").Delete(&model.RecordingPolicyDevice{}).Error; err != nil {
			return err
		}
		if mode == model.RecordingModeSelected {
			for peerId := range clean {
				if err := tx.Create(&model.RecordingPolicyDevice{PeerId: peerId}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func recordingExtension(filename string) (string, string, error) {
	name := filepath.Base(strings.TrimSpace(filename))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '/' || r == '\\' || r == '"' {
			return '_'
		}
		return r
	}, name)
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".webm":
		return name, "webm", nil
	case ".mp4":
		return name, "mp4", nil
	default:
		return "", "", errors.New("only .webm and .mp4 recordings are accepted")
	}
}

func (s *RecordingService) InitUpload(in *RecordingInit) (*model.SessionRecording, string, error) {
	peer := AllService.PeerService.FindById(in.PeerId)
	if peer.RowId == 0 || peer.Uuid == "" || peer.Uuid != in.Uuid {
		return nil, "", errors.New("device identity does not match")
	}
	enabled, err := s.IsEnabled(in.PeerId)
	if err != nil {
		return nil, "", err
	}
	if !enabled {
		return nil, "", errors.New("recording is disabled for this device")
	}
	name, container, err := recordingExtension(in.Filename)
	if err != nil {
		return nil, "", err
	}
	uploadId := strings.TrimSpace(in.UploadId)
	token := strings.TrimSpace(in.UploadToken)
	if uploadId != "" {
		if _, err := uuid.Parse(uploadId); err != nil || len(token) < 32 {
			return nil, "", errors.New("invalid client upload identity")
		}
		existing := &model.SessionRecording{}
		err := DB.Where("upload_id = ?", uploadId).First(existing).Error
		if err == nil {
			if existing.PeerId != in.PeerId || subtle.ConstantTimeCompare([]byte(tokenHash(token)), []byte(existing.UploadTokenHash)) != 1 {
				return nil, "", errors.New("upload identity is already in use")
			}
			return existing, token, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", err
		}
	} else {
		uploadId = uuid.NewString()
		var err error
		token, err = randomToken()
		if err != nil {
			return nil, "", err
		}
	}
	storageName := uploadId + "." + container
	storageSettingId, storageConfig, err := s.activeStorageConfig()
	if err != nil {
		return nil, "", err
	}
	if err = os.MkdirAll(s.storagePath(), 0750); err != nil {
		return nil, "", err
	}
	file, err := os.OpenFile(filepath.Join(s.storagePath(), storageName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return nil, "", err
	}
	if err = file.Close(); err != nil {
		return nil, "", err
	}
	startedAt := in.StartedAt
	if startedAt <= 0 {
		startedAt = time.Now().Unix()
	}
	recording := &model.SessionRecording{
		UploadId: uploadId, UploadTokenHash: tokenHash(token), PeerId: in.PeerId,
		FromPeer: in.FromPeer, FromName: in.FromName, SessionId: in.SessionId, OriginalName: name,
		StorageName: storageName, StorageBackend: storageConfig.Backend, StorageSettingId: storageSettingId,
		Container: container, Codec: strings.ToLower(in.Codec),
		Status: model.RecordingStatusUploading, StartedAt: startedAt,
	}
	if err = DB.Create(recording).Error; err != nil {
		_ = os.Remove(filepath.Join(s.storagePath(), storageName))
		return nil, "", err
	}
	return recording, token, nil
}

func (s *RecordingService) Authorized(uploadId, token string) (*model.SessionRecording, error) {
	recording := &model.SessionRecording{}
	if err := DB.Where("upload_id = ?", uploadId).First(recording).Error; err != nil {
		return nil, err
	}
	providedHash := tokenHash(token)
	if token == "" || subtle.ConstantTimeCompare([]byte(providedHash), []byte(recording.UploadTokenHash)) != 1 {
		return nil, errors.New("invalid upload token")
	}
	return recording, nil
}

func (s *RecordingService) WriteChunk(recording *model.SessionRecording, offset int64, body io.Reader, length int64) (int64, error) {
	if recording.Status != model.RecordingStatusUploading {
		return 0, errors.New("upload is not active")
	}
	if offset < 0 || length < 0 || length > s.MaxChunkSize() {
		return 0, errors.New("invalid chunk offset or size")
	}
	file, err := os.OpenFile(filepath.Join(s.storagePath(), recording.StorageName), os.O_WRONLY, 0640)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	written, err := io.CopyN(io.NewOffsetWriter(file, offset), body, length)
	if err != nil {
		return written, err
	}
	end := offset + written
	if end > recording.Size {
		recording.Size = end
		if err = DB.Model(recording).Update("size", end).Error; err != nil {
			return written, err
		}
	}
	return written, nil
}

func (s *RecordingService) Complete(recording *model.SessionRecording, durationMs int64, expectedHash string) error {
	if recording.Status == model.RecordingStatusComplete || recording.Status == model.RecordingStatusTranscoding {
		if expectedHash == "" || strings.EqualFold(expectedHash, recording.Sha256) {
			return nil
		}
		return errors.New("sha256 mismatch for completed recording")
	}
	if recording.Status != model.RecordingStatusUploading {
		return errors.New("upload is not active")
	}
	path := filepath.Join(s.storagePath(), recording.StorageName)
	if recording.Size <= 0 {
		return errors.New("recording is empty")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	h := sha256.New()
	if _, err = io.Copy(h, file); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	hash := hex.EncodeToString(h.Sum(nil))
	if expectedHash != "" && !strings.EqualFold(expectedHash, hash) {
		return fmt.Errorf("sha256 mismatch")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	status := model.RecordingStatusComplete
	if recording.Codec == "h265" || recording.Codec == "hevc" {
		status = model.RecordingStatusTranscoding
	}
	if err = s.archiveRecordingObject(recording, recording.StorageName, path); err != nil {
		return fmt.Errorf("archive recording: %w", err)
	}
	if err = DB.Model(recording).Updates(map[string]interface{}{
		"status": status, "size": info.Size(), "duration_ms": durationMs,
		"completed_at": time.Now().Unix(), "sha256": hash,
	}).Error; err != nil {
		return err
	}
	if status == model.RecordingStatusTranscoding {
		go s.TranscodePreview(recording.Id)
	} else {
		s.removeStagedRecordingObject(recording, recording.StorageName)
	}
	return nil
}

func (s *RecordingService) ffmpegPath() string {
	path := strings.TrimSpace(Config.Recording.FfmpegPath)
	if path == "" {
		return "ffmpeg"
	}
	return path
}

func (s *RecordingService) TranscodePreview(id uint) {
	recording, err := s.Info(id)
	if err != nil || recording.Status != model.RecordingStatusTranscoding {
		return
	}
	previewName := recording.UploadId + ".preview.mp4"
	previewPath := filepath.Join(s.storagePath(), previewName)
	originalPath := s.FilePath(recording)
	originalCleanup := func() {}
	if _, statErr := os.Stat(originalPath); errors.Is(statErr, os.ErrNotExist) {
		originalPath, originalCleanup, err = s.MaterializeRecordingObject(recording, false)
		if err != nil {
			_ = DB.Model(recording).Updates(map[string]interface{}{
				"status": model.RecordingStatusFailed, "error_message": "preview source unavailable: " + err.Error(),
			}).Error
			return
		}
	}
	defer originalCleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.ffmpegPath(), "-hide_banner", "-loglevel", "error", "-y",
		"-i", originalPath, "-map", "0:v:0", "-an", "-c:v", "libx264",
		"-preset", "veryfast", "-crf", "28", "-movflags", "+faststart", previewPath)
	output, runErr := cmd.CombinedOutput()
	if runErr != nil {
		_ = os.Remove(previewPath)
		message := strings.TrimSpace(string(output))
		if len(message) > 500 {
			message = message[len(message)-500:]
		}
		if message == "" {
			message = runErr.Error()
		}
		_ = DB.Model(recording).Updates(map[string]interface{}{
			"status": model.RecordingStatusFailed, "error_message": "preview transcoding failed: " + message,
		}).Error
		s.removeStagedRecordingObject(recording, recording.StorageName)
		return
	}
	if err = s.archiveRecordingObject(recording, previewName, previewPath); err != nil {
		_ = DB.Model(recording).Updates(map[string]interface{}{
			"status": model.RecordingStatusFailed, "error_message": "preview archive failed: " + err.Error(),
		}).Error
		return
	}
	result := DB.Model(recording).Where("status = ?", model.RecordingStatusTranscoding).Updates(map[string]interface{}{
		"status": model.RecordingStatusComplete, "preview_storage_name": previewName, "error_message": "",
	})
	if result.Error != nil || result.RowsAffected == 0 {
		_ = os.Remove(previewPath)
		return
	}
	s.removeStagedRecordingObject(recording, recording.StorageName)
	s.removeStagedRecordingObject(recording, previewName)
}

func (s *RecordingService) List(page, pageSize uint, peerId, fromPeer, status string, startedAfter, startedBefore int64) *model.SessionRecordingList {
	res := &model.SessionRecordingList{}
	res.Page, res.PageSize = int64(page), int64(pageSize)
	tx := DB.Model(&model.SessionRecording{})
	if peerId != "" {
		tx = tx.Where("peer_id like ?", "%"+peerId+"%")
	}
	if fromPeer != "" {
		term := "%" + fromPeer + "%"
		tx = tx.Where("from_peer like ? OR from_name like ?", term, term)
	}
	if status != "" {
		tx = tx.Where("status = ?", status)
	}
	if startedAfter > 0 {
		tx = tx.Where("started_at >= ?", startedAfter)
	}
	if startedBefore > 0 {
		tx = tx.Where("started_at <= ?", startedBefore)
	}
	tx.Count(&res.Total)
	tx.Order("id desc").Scopes(Paginate(page, pageSize)).Find(&res.Recordings)
	return res
}

func (s *RecordingService) Info(id uint) (*model.SessionRecording, error) {
	recording := &model.SessionRecording{}
	return recording, DB.First(recording, id).Error
}

func (s *RecordingService) FilePath(recording *model.SessionRecording) string {
	return filepath.Join(s.storagePath(), recording.StorageName)
}

func (s *RecordingService) PreviewFilePath(recording *model.SessionRecording) string {
	if recording.PreviewStorageName == "" {
		return s.FilePath(recording)
	}
	return filepath.Join(s.storagePath(), recording.PreviewStorageName)
}

func (s *RecordingService) Delete(recording *model.SessionRecording) error {
	if err := s.deleteRecordingObject(recording, recording.StorageName); err != nil {
		return err
	}
	_ = os.Remove(s.FilePath(recording))
	if recording.PreviewStorageName != "" {
		if err := s.deleteRecordingObject(recording, recording.PreviewStorageName); err != nil {
			return err
		}
		_ = os.Remove(s.PreviewFilePath(recording))
	}
	return DB.Delete(recording).Error
}

func (s *RecordingService) Cancel(recording *model.SessionRecording) error {
	if recording.Status != model.RecordingStatusUploading {
		return errors.New("upload is not active")
	}
	return s.Delete(recording)
}

func (s *RecordingService) DeleteMany(ids []uint) error {
	if len(ids) == 0 {
		return errors.New("recording ids are required")
	}
	var recordings []*model.SessionRecording
	if err := DB.Where("id IN ?", ids).Find(&recordings).Error; err != nil {
		return err
	}
	for _, recording := range recordings {
		if err := s.Delete(recording); err != nil {
			return err
		}
	}
	return nil
}

func (s *RecordingService) CleanupExpired(now time.Time) error {
	policy, _, err := s.Policy()
	if err != nil {
		return err
	}
	cutoff := now.Add(-time.Duration(policy.RetentionDays) * 24 * time.Hour).Unix()
	var expired []*model.SessionRecording
	if err = DB.Where("status IN ? AND ((completed_at > 0 AND completed_at < ?) OR (completed_at = 0 AND started_at < ?))",
		[]string{model.RecordingStatusComplete, model.RecordingStatusFailed}, cutoff, cutoff).
		Find(&expired).Error; err != nil {
		return err
	}
	for _, recording := range expired {
		if err = s.Delete(recording); err != nil {
			Logger.Warnf("failed to delete expired recording %d: %v", recording.Id, err)
		}
	}
	staleCutoff := now.Add(-24 * time.Hour).Unix()
	return DB.Model(&model.SessionRecording{}).
		Where("status = ? AND started_at < ?", model.RecordingStatusUploading, staleCutoff).
		Updates(map[string]interface{}{
			"status": model.RecordingStatusFailed, "error_message": "upload timed out",
		}).Error
}

func StartRecordingMaintenance() {
	run := func() {
		if err := AllService.RecordingService.CleanupExpired(time.Now()); err != nil {
			Logger.Warnf("recording maintenance failed: %v", err)
		}
	}
	run()
	var pending []*model.SessionRecording
	if err := DB.Where("status = ?", model.RecordingStatusTranscoding).Find(&pending).Error; err == nil {
		for _, recording := range pending {
			go AllService.RecordingService.TranscodePreview(recording.Id)
		}
	}
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}
