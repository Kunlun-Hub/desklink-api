package service

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestRecordingStorageCredentialsAreEncrypted(t *testing.T) {
	setupRecordingTest(t)
	value := recordingStorageSecrets{SecretKey: "s3-secret", Password: "ftp-password"}
	encrypted, err := encryptRecordingStorageSecrets(value)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "" || bytes.Contains([]byte(encrypted), []byte(value.SecretKey)) || bytes.Contains([]byte(encrypted), []byte(value.Password)) {
		t.Fatal("recording storage credentials were not encrypted")
	}
	decrypted, err := decryptRecordingStorageSecrets(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != value {
		t.Fatalf("unexpected decrypted credentials: %#v", decrypted)
	}
}

func TestMountedRecordingStorageLifecycle(t *testing.T) {
	service := setupRecordingTest(t)
	mountPath := t.TempDir()
	config, err := service.SaveStorageConfig(RecordingStorageConfig{
		Backend: recordingStorageNFS, Path: mountPath, Prefix: "desklink/recordings",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.Backend != recordingStorageNFS {
		t.Fatalf("unexpected active storage config: %#v", config)
	}
	if err = DB.Create(&model.Peer{Id: "peer-a", Uuid: "uuid-a"}).Error; err != nil {
		t.Fatal(err)
	}
	if err = service.SavePolicy(model.RecordingModeAll, 30, nil); err != nil {
		t.Fatal(err)
	}
	recording, _, err := service.InitUpload(&RecordingInit{PeerId: "peer-a", Uuid: "uuid-a", Filename: "mounted.webm", Codec: "vp9"})
	if err != nil {
		t.Fatal(err)
	}
	if recording.StorageBackend != recordingStorageNFS {
		t.Fatalf("recording did not retain its storage backend: %#v", recording)
	}
	if _, err = service.WriteChunk(recording, 0, bytes.NewReader([]byte("mounted-data")), 12); err != nil {
		t.Fatal(err)
	}
	if err = service.Complete(recording, 1000, ""); err != nil {
		t.Fatal(err)
	}
	archived := filepath.Join(mountPath, "desklink", "recordings", recording.StorageName)
	content, err := os.ReadFile(archived)
	if err != nil || string(content) != "mounted-data" {
		t.Fatalf("unexpected mounted storage content %q: %v", content, err)
	}
	if _, err = os.Stat(service.FilePath(recording)); !os.IsNotExist(err) {
		t.Fatalf("external recording staging file was not removed: %v", err)
	}
	secondMount := t.TempDir()
	if _, err = service.SaveStorageConfig(RecordingStorageConfig{
		Backend: recordingStorageNFS, Path: secondMount, Prefix: "new-recordings",
	}); err != nil {
		t.Fatal(err)
	}
	materialized, cleanup, err := service.MaterializeRecordingObject(recording, false)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	content, err = os.ReadFile(materialized)
	if err != nil || string(content) != "mounted-data" {
		t.Fatalf("unexpected materialized content %q: %v", content, err)
	}
	if _, err = os.Stat(filepath.Join(secondMount, "new-recordings", recording.StorageName)); !os.IsNotExist(err) {
		t.Fatalf("old recording followed the newly activated storage config: %v", err)
	}
	if err = service.Delete(recording); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(archived); !os.IsNotExist(err) {
		t.Fatalf("mounted recording was not deleted: %v", err)
	}
}

func TestRecordingObjectStoreIntegration(t *testing.T) {
	tests := []struct {
		name   string
		config RecordingStorageConfig
	}{
		{name: "ftp", config: RecordingStorageConfig{
			Backend: recordingStorageFTP, Endpoint: os.Getenv("DESKLINK_TEST_FTP_ENDPOINT"),
			Username: os.Getenv("DESKLINK_TEST_FTP_USERNAME"), Password: os.Getenv("DESKLINK_TEST_FTP_PASSWORD"),
			Prefix: "desklink-test",
		}},
		{name: "s3", config: RecordingStorageConfig{
			Backend: recordingStorageS3, Endpoint: os.Getenv("DESKLINK_TEST_S3_ENDPOINT"),
			Bucket: "desklink-test", AccessKey: os.Getenv("DESKLINK_TEST_S3_ACCESS_KEY"),
			SecretKey: os.Getenv("DESKLINK_TEST_S3_SECRET_KEY"), Prefix: "recordings",
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.config.Endpoint == "" {
				t.Skip("external recording storage test endpoint is not configured")
			}
			service := setupRecordingTest(t)
			store, err := service.newRecordingObjectStore(test.config)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if err = store.Check(ctx); err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(t.TempDir(), "source.webm")
			if err = os.WriteFile(source, []byte("external-storage"), 0600); err != nil {
				t.Fatal(err)
			}
			if err = store.Archive(ctx, source, "session.webm"); err != nil {
				t.Fatal(err)
			}
			value, cleanup, err := store.Materialize(ctx, "session.webm")
			if err != nil {
				t.Fatal(err)
			}
			file, err := os.Open(value)
			if err != nil {
				t.Fatal(err)
			}
			content, readErr := io.ReadAll(file)
			_ = file.Close()
			cleanup()
			if readErr != nil || string(content) != "external-storage" {
				t.Fatalf("unexpected external storage content %q: %v", content, readErr)
			}
			if err = store.Delete(ctx, "session.webm"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRecordingSessionMerge(t *testing.T) {
	service := setupRecordingTest(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if err := DB.Create(&model.Peer{Id: "peer-merge", Uuid: "uuid-merge"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.SavePolicy(model.RecordingModeAll, 30, nil); err != nil {
		t.Fatal(err)
	}
	segmentDir := t.TempDir()
	makeSegment := func(name, color string) string {
		path := filepath.Join(segmentDir, name)
		command := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c="+color+":s=320x240:r=10", "-t", "1", "-an", "-c:v", "libx264", "-pix_fmt", "yuv420p", path)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("create test segment: %v: %s", err, output)
		}
		return path
	}
	firstPath := makeSegment("first.mp4", "red")
	secondPath := makeSegment("second.mp4", "blue")
	firstInfo, err := os.Stat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	secondInfo, err := os.Stat(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	first := &model.SessionRecording{
		UploadId: "merge-first", UploadTokenHash: "unused", PeerId: "peer-merge", FromPeer: "controller",
		FromName: "Admin", SessionId: "session-merge", OriginalName: "first.mp4", StorageName: "first.mp4",
		StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete,
		Size: firstInfo.Size(), DurationMs: 1000,
	}
	second := &model.SessionRecording{
		UploadId: "merge-second", UploadTokenHash: "unused", PeerId: "peer-merge", FromPeer: "controller",
		FromName: "Admin", SessionId: "session-merge", OriginalName: "second.mp4", StorageName: "second.mp4",
		StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete,
		Size: secondInfo.Size(), DurationMs: 1000,
	}
	if err = os.Rename(firstPath, service.FilePath(first)); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(secondPath, service.FilePath(second)); err != nil {
		t.Fatal(err)
	}
	if err = DB.Create([]*model.SessionRecording{first, second}).Error; err != nil {
		t.Fatal(err)
	}
	if err = service.MergeSession(first); err != nil {
		t.Fatal(err)
	}
	var records []*model.SessionRecording
	if err = DB.Where("session_id = ?", "session-merge").Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Id != first.Id || records[0].DurationMs != 2000 || records[0].Container != "mp4" {
		t.Fatalf("unexpected merged metadata: %#v", records)
	}
	mergedPath := service.FilePath(records[0])
	mergedInfo, err := os.Stat(mergedPath)
	if err != nil || mergedInfo.Size() <= firstInfo.Size() {
		t.Fatalf("merged recording is missing or too small: %v", err)
	}
	if _, err = os.Stat(filepath.Join(service.storagePath(), "second.mp4")); !os.IsNotExist(err) {
		t.Fatalf("duplicate segment still exists: %v", err)
	}
}

func TestRecordingSessionMergeWithRelativeStoragePath(t *testing.T) {
	service := setupRecordingTest(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	storageDirectory, err := os.MkdirTemp(workingDirectory, ".recording-relative-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(storageDirectory)
	relativePath, err := filepath.Rel(workingDirectory, storageDirectory)
	if err != nil {
		t.Fatal(err)
	}
	Config.Recording.Path = relativePath
	makeSegment := func(name, color string) {
		path := filepath.Join(storageDirectory, name)
		command := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c="+color+":s=160x120:r=10", "-t", "1", "-an", "-c:v", "libx264", "-pix_fmt", "yuv420p", path)
		if output, commandErr := command.CombinedOutput(); commandErr != nil {
			t.Fatalf("create relative test segment: %v: %s", commandErr, output)
		}
	}
	makeSegment("relative-first.mp4", "black")
	makeSegment("relative-second.mp4", "white")
	segments := []*model.SessionRecording{
		{UploadId: "relative-first", UploadTokenHash: "unused", PeerId: "relative-peer", FromPeer: "controller", SessionId: "relative-session", OriginalName: "relative-first.mp4", StorageName: "relative-first.mp4", StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete, DurationMs: 1000},
		{UploadId: "relative-second", UploadTokenHash: "unused", PeerId: "relative-peer", FromPeer: "controller", SessionId: "relative-session", OriginalName: "relative-second.mp4", StorageName: "relative-second.mp4", StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete, DurationMs: 1000},
	}
	for _, segment := range segments {
		info, statErr := os.Stat(filepath.Join(storageDirectory, segment.StorageName))
		if statErr != nil {
			t.Fatal(statErr)
		}
		segment.Size = info.Size()
	}
	if err = DB.Create(&segments).Error; err != nil {
		t.Fatal(err)
	}
	if err = service.MergeSession(segments[0]); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = DB.Model(&model.SessionRecording{}).Where("session_id = ?", "relative-session").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("relative storage session was not merged: %d", count)
	}
}

func TestRecordingSessionMergeReencodesMixedCodecs(t *testing.T) {
	service := setupRecordingTest(t)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	create := func(name, codec, color, size string) {
		args := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=" + color + ":s=" + size + ":r=10", "-t", "1", "-an", "-c:v", codec}
		if codec == "libx264" {
			args = append(args, "-pix_fmt", "yuv420p")
		}
		args = append(args, filepath.Join(service.storagePath(), name))
		if output, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
			t.Fatalf("create mixed codec segment: %v: %s", err, output)
		}
	}
	create("mixed-first.mp4", "libx264", "red", "320x240")
	create("mixed-second.webm", "libvpx-vp9", "blue", "640x360")
	segments := []*model.SessionRecording{
		{UploadId: "mixed-first", UploadTokenHash: "unused", PeerId: "mixed-peer", FromPeer: "controller", SessionId: "mixed-session", OriginalName: "mixed-first.mp4", StorageName: "mixed-first.mp4", StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete},
		{UploadId: "mixed-second", UploadTokenHash: "unused", PeerId: "mixed-peer", FromPeer: "controller", SessionId: "mixed-session", OriginalName: "mixed-second.webm", StorageName: "mixed-second.webm", StorageBackend: recordingStorageLocal, Container: "webm", Codec: "vp9", Status: model.RecordingStatusComplete},
	}
	for _, segment := range segments {
		info, err := os.Stat(service.FilePath(segment))
		if err != nil {
			t.Fatal(err)
		}
		segment.Size = info.Size()
	}
	if err := DB.Create(&segments).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.MergeSession(segments[0]); err != nil {
		t.Fatal(err)
	}
	merged, err := service.Info(segments[0].Id)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Codec != "h264" || merged.Container != "mp4" || merged.DurationMs < 1900 {
		t.Fatalf("unexpected mixed-codec merge: %#v", merged)
	}
}

func TestMergeExistingSessionsFindsSessionID(t *testing.T) {
	service := setupRecordingTest(t)
	first := &model.SessionRecording{
		UploadId: "scan-first", UploadTokenHash: "unused", PeerId: "scan-peer", FromPeer: "scan-controller",
		SessionId: "scan-session", OriginalName: "first.mp4", StorageName: "scan-first.mp4",
		StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete,
	}
	second := &model.SessionRecording{
		UploadId: "scan-second", UploadTokenHash: "unused", PeerId: "scan-peer", FromPeer: "scan-controller",
		SessionId: "scan-session", OriginalName: "second.mp4", StorageName: "scan-second.mp4",
		StorageBackend: recordingStorageLocal, Container: "mp4", Codec: "h264", Status: model.RecordingStatusComplete,
	}
	if err := DB.Create([]*model.SessionRecording{first, second}).Error; err != nil {
		t.Fatal(err)
	}
	keys, err := service.mergeableSessionKeys()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].PeerId != "scan-peer" || keys[0].FromPeer != "scan-controller" || keys[0].Session != "scan-session" {
		t.Fatalf("unexpected mergeable session keys: %#v", keys)
	}
}
