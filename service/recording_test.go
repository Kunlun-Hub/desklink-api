package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRenderCursorRecordingFile(t *testing.T) {
	service := setupRecordingTest(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	source := filepath.Join(service.storagePath(), "cursor-source.mp4")
	if output, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=white:s=320x240:r=10", "-t", "1", "-an",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", source).CombinedOutput(); err != nil {
		t.Fatalf("create cursor source: %v: %s", err, output)
	}
	recording := &model.SessionRecording{
		UploadId: "cursor-render", UploadTokenHash: "unused", PeerId: "peer", OriginalName: "recording.mp4",
		StorageName: "cursor-source.mp4", StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264",
		Status: model.RecordingStatusComplete, DurationMs: 1000,
		CursorTrack: `[{"t":0,"x":1000,"y":1000,"visible":true},{"t":500,"x":50000,"y":40000,"visible":true}]`,
	}
	if err := DB.Create(recording).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.renderCursorRecordingFile(recording); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Info(recording.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.CursorRenderStatus != recordingCursorReady || stored.CursorStorageName == "" {
		t.Fatalf("cursor recording was not marked ready: %#v", stored)
	}
	path, cleanup, err := service.MaterializeCursorRecording(stored)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatalf("cursor recording is missing: %v", err)
	}
	if output, err := exec.Command("ffmpeg", "-v", "error", "-i", path, "-map", "0:v:0", "-f", "null", "-").CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("cursor recording did not decode cleanly: %v: %s", err, output)
	}
	if err := service.Delete(stored); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(service.storagePath(), stored.CursorStorageName)); !os.IsNotExist(err) {
		t.Fatalf("cursor recording remains after deletion: %v", err)
	}
}

func TestRecordingAccessTokenSeparatesCursorDownload(t *testing.T) {
	setupRecordingTest(t)
	token, _, err := CreateRecordingAccessToken(42, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyRecordingAccessToken(token, 42, true, true) {
		t.Fatal("cursor download token should be valid for its intended use")
	}
	if VerifyRecordingAccessToken(token, 42, true, false) || VerifyRecordingAccessToken(token, 42, false, false) {
		t.Fatal("cursor download token was accepted for a different content type")
	}
}

func setupRecordingTest(t *testing.T) *RecordingService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(
		&model.Peer{}, &model.RecordingPolicy{}, &model.RecordingPolicyDevice{},
		&model.SessionRecording{}, &model.RecordingStorageSetting{},
	); err != nil {
		t.Fatal(err)
	}
	DB = db
	Config = &config.Config{
		Jwt:       config.Jwt{Key: "recording-storage-test-key"},
		Recording: config.Recording{Path: t.TempDir(), MaxChunkSize: 8 * 1024 * 1024},
	}
	Logger = logrus.New()
	service := &RecordingService{}
	AllService = &Service{PeerService: &PeerService{}, RecordingService: service}
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return service
}

func TestRecordingPolicyModes(t *testing.T) {
	service := setupRecordingTest(t)
	Config.Recording.MaxChunkSize = 1
	if service.MaxChunkSize() != minRecordingChunkSize {
		t.Fatal("recording chunk size was not clamped to the minimum")
	}
	Config.Recording.MaxChunkSize = maxRecordingChunkSize + 1
	if service.MaxChunkSize() != maxRecordingChunkSize {
		t.Fatal("recording chunk size was not clamped to the maximum")
	}

	if enabled, err := service.IsEnabled("peer-a"); err != nil || enabled {
		t.Fatalf("default policy should be off: enabled=%v err=%v", enabled, err)
	}
	if err := service.SavePolicy(model.RecordingModeSelected, 30, []string{"peer-a", "peer-a", "peer-b"}); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := service.IsEnabled("peer-a"); !enabled {
		t.Fatal("selected peer should be enabled")
	}
	if enabled, _ := service.IsEnabled("peer-c"); enabled {
		t.Fatal("unselected peer should be disabled")
	}
	if err := service.SavePolicy(model.RecordingModeAll, 30, nil); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := service.IsEnabled("peer-c"); !enabled {
		t.Fatal("all policy should enable every peer")
	}
	if err := service.SavePolicy("invalid", 30, nil); err == nil {
		t.Fatal("invalid policy mode was accepted")
	}
}

func TestRecordingUploadLifecycle(t *testing.T) {
	service := setupRecordingTest(t)
	if err := DB.Create(&model.Peer{Id: "peer-a", Uuid: "uuid-a"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SavePolicy(model.RecordingModeSelected, 30, []string{"peer-a"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.InitUpload(&RecordingInit{PeerId: "peer-a", Uuid: "wrong", Filename: "session.webm"}); err == nil {
		t.Fatal("mismatched device identity was accepted")
	}
	recording, token, err := service.InitUpload(&RecordingInit{
		PeerId: "peer-a", Uuid: "uuid-a", FromPeer: "operator-a", SessionId: "session-a",
		Filename: "../session\r\n.webm", Codec: "vp9", StartedAt: time.Now().Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if recording.OriginalName != "session__.webm" {
		t.Fatalf("filename was not sanitized: %q", recording.OriginalName)
	}
	if _, err = service.Authorized(recording.UploadId, "wrong"); err == nil {
		t.Fatal("invalid upload token was accepted")
	}
	authorized, err := service.Authorized(recording.UploadId, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.WriteChunk(authorized, 4, bytes.NewReader([]byte("efgh")), 4); err != nil {
		t.Fatal(err)
	}
	if _, err = service.WriteChunk(authorized, 0, bytes.NewReader([]byte("abcd")), 4); err != nil {
		t.Fatal(err)
	}
	// A repeated chunk after a lost response must be idempotent.
	if _, err = service.WriteChunk(authorized, 0, bytes.NewReader([]byte("abcd")), 4); err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte("abcdefgh"))
	cursorTrack := []RecordingCursorSample{{Time: 100, X: 1200, Y: 2400, Visible: true}}
	if err = service.Complete(authorized, 1234, hex.EncodeToString(expected[:]), cursorTrack); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Info(recording.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != model.RecordingStatusComplete || stored.Size != 8 || stored.DurationMs != 1234 {
		t.Fatalf("unexpected completed recording: %+v", stored)
	}
	if stored.CursorTrack != `[{"t":100,"x":1200,"y":2400,"visible":true}]` {
		t.Fatalf("unexpected cursor track: %s", stored.CursorTrack)
	}
	authorized, err = service.Authorized(recording.UploadId, token)
	if err != nil {
		t.Fatal("completed upload token should remain valid for idempotent completion")
	}
	if err = service.Complete(authorized, 1234, hex.EncodeToString(expected[:]), nil); err != nil {
		t.Fatalf("repeated completion was not idempotent: %v", err)
	}
	file, err := os.Open(service.FilePath(stored))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil || string(content) != "abcdefgh" {
		t.Fatalf("unexpected recording content %q: %v", content, err)
	}
	if _, err = service.WriteChunk(authorized, 0, bytes.NewReader([]byte("xxxx")), 4); err == nil {
		t.Fatal("completed recording accepted another chunk")
	}
}

func TestRecordingCancel(t *testing.T) {
	service := setupRecordingTest(t)
	if err := DB.Create(&model.Peer{Id: "peer-a", Uuid: "uuid-a"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SavePolicy(model.RecordingModeAll, 30, nil); err != nil {
		t.Fatal(err)
	}
	recording, _, err := service.InitUpload(&RecordingInit{PeerId: "peer-a", Uuid: "uuid-a", Filename: "short.webm"})
	if err != nil {
		t.Fatal(err)
	}
	path := service.FilePath(recording)
	if err = service.Cancel(recording); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cancelled upload file still exists: %v", err)
	}
	if err = DB.First(&model.SessionRecording{}, recording.Id).Error; err == nil {
		t.Fatal("cancelled upload metadata still exists")
	}
}

func TestRecordingInitIsIdempotent(t *testing.T) {
	service := setupRecordingTest(t)
	if err := DB.Create(&model.Peer{Id: "peer-a", Uuid: "uuid-a"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SavePolicy(model.RecordingModeAll, 30, nil); err != nil {
		t.Fatal(err)
	}
	form := &RecordingInit{
		UploadId: uuid.NewString(), UploadToken: "0123456789abcdef0123456789abcdef",
		PeerId: "peer-a", Uuid: "uuid-a", Filename: "retry.webm",
	}
	first, token, err := service.InitUpload(form)
	if err != nil {
		t.Fatal(err)
	}
	second, secondToken, err := service.InitUpload(form)
	if err != nil {
		t.Fatal(err)
	}
	if first.Id != second.Id || token != secondToken {
		t.Fatal("repeated init created a second upload")
	}
	var count int64
	DB.Model(&model.SessionRecording{}).Count(&count)
	if count != 1 {
		t.Fatalf("expected one upload, got %d", count)
	}
}

func TestRecordingCleanup(t *testing.T) {
	service := setupRecordingTest(t)
	if err := service.SavePolicy(model.RecordingModeOff, 1, nil); err != nil {
		t.Fatal(err)
	}
	oldFile := "old.webm"
	if err := os.WriteFile(service.storagePath()+"/"+oldFile, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	old := &model.SessionRecording{
		UploadId: "old-upload", UploadTokenHash: "unused", PeerId: "peer-a", OriginalName: oldFile,
		StorageName: oldFile, Container: "webm", Status: model.RecordingStatusComplete,
		StartedAt: time.Now().Add(-72 * time.Hour).Unix(), CompletedAt: time.Now().Add(-48 * time.Hour).Unix(),
	}
	stale := &model.SessionRecording{
		UploadId: "stale-upload", UploadTokenHash: "unused", PeerId: "peer-a", OriginalName: "stale.webm",
		StorageName: "stale.webm", Container: "webm", Status: model.RecordingStatusUploading,
		StartedAt: time.Now().Add(-48 * time.Hour).Unix(),
	}
	if err := DB.Create(old).Error; err != nil {
		t.Fatal(err)
	}
	if err := DB.Create(stale).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.CleanupExpired(time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(service.storagePath() + "/" + oldFile); !os.IsNotExist(err) {
		t.Fatalf("expired file still exists: %v", err)
	}
	if err := DB.First(&model.SessionRecording{}, old.Id).Error; err == nil {
		t.Fatal("expired metadata still exists")
	}
	if err := DB.First(stale, stale.Id).Error; err != nil {
		t.Fatal(err)
	}
	if stale.Status != model.RecordingStatusFailed {
		t.Fatalf("stale upload was not marked failed: %s", stale.Status)
	}
}
