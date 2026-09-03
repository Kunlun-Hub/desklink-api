package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
)

func TestCreateUserNormalizesUniqueUsernameAndEncryptsPassword(t *testing.T) {
	setupRecordingTest(t)
	if err := DB.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	AllService.UserService = &UserService{}
	if err := AllService.UserService.Create(&model.User{Username: "empty-password", GroupId: 1, Status: model.COMMON_STATUS_ENABLE}); err == nil {
		t.Fatal("user without a password was accepted")
	}
	AllService.LdapService = &LdapService{}
	user := &model.User{Username: "Admin User", Password: "secret123", GroupId: 1, Status: model.COMMON_STATUS_ENABLE}
	if err := AllService.UserService.Create(user); err != nil {
		t.Fatal(err)
	}
	if user.Username != "adminuser" || user.Password == "secret123" {
		t.Fatalf("user was not normalized/encrypted: %#v", user)
	}
	ok, _, err := utils.VerifyPassword(user.Password, "secret123")
	if err != nil || !ok {
		t.Fatalf("stored password cannot be verified: %v", err)
	}
	duplicate := &model.User{Username: "ADMIN USER", Password: "different", GroupId: 1, Status: model.COMMON_STATUS_ENABLE}
	if err := AllService.UserService.Create(duplicate); err == nil {
		t.Fatal("case/space-equivalent username was accepted")
	}
}

func TestAuditConnListHidesCoordinationConnections(t *testing.T) {
	setupRecordingTest(t)
	if err := DB.AutoMigrate(&model.AuditConn{}); err != nil {
		t.Fatal(err)
	}
	AllService.AuditService = &AuditService{}
	records := []*model.AuditConn{
		{Action: model.AuditActionNew, PeerId: "target", Ip: "10.0.0.1", SessionId: "0"},
		{Action: model.AuditActionNew, PeerId: "target", Ip: "10.0.0.1", SessionId: "", FromPeer: ""},
		{Action: model.AuditActionNew, PeerId: "target", FromPeer: "source", FromName: "Admin", Ip: "10.0.0.1", SessionId: "session-1"},
	}
	if err := DB.Create(&records).Error; err != nil {
		t.Fatal(err)
	}
	result := AllService.AuditService.AuditConnList(1, 20, nil)
	if result.Total != 1 || len(result.AuditConns) != 1 || result.AuditConns[0].SessionId != "session-1" {
		t.Fatalf("unexpected connection audit list: %#v", result)
	}
}
