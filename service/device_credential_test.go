package service

import (
	"github.com/lejianwen/rustdesk-api/v2/model"
	"testing"
)

func TestDeviceCredentialsAreEncryptedAndRecoverable(t *testing.T) {
	setupRecordingTest(t)
	if err := DB.AutoMigrate(&model.DeviceCredential{}); err != nil {
		t.Fatal(err)
	}
	temporary := "temporary-123"
	permanent := "permanent-456"
	if err := UpsertDeviceCredentials("peer-credentials", &temporary, &permanent); err != nil {
		t.Fatal(err)
	}
	var stored model.DeviceCredential
	if err := DB.First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.EncryptedSecret == "" || stored.EncryptedSecret == temporary || stored.EncryptedSecret == permanent {
		t.Fatalf("credentials were not encrypted: %q", stored.EncryptedSecret)
	}
	secrets, status, err := DeviceCredentials("peer-credentials")
	if err != nil || secrets.TemporaryPassword != temporary || secrets.PermanentPassword != permanent || !status.HasTemporary || !status.HasPermanent {
		t.Fatalf("unexpected credentials: %#v %#v %v", secrets, status, err)
	}
}
