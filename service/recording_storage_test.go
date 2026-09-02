package service

import (
	"bytes"
	"context"
	"io"
	"os"
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
