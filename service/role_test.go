package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRolePermissionsAndBuiltIns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:role-permissions-test?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Role{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldServices := DB, AllService
	DB = db
	AllService = &Service{UserService: &UserService{}, RoleService: &RoleService{}}
	t.Cleanup(func() {
		DB, AllService = oldDB, oldServices
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	legacyAdmin := &model.User{Username: "legacy-admin", IsAdmin: boolPtr(true), Status: model.COMMON_STATUS_ENABLE}
	if err = db.Create(legacyAdmin).Error; err != nil {
		t.Fatal(err)
	}
	if err = AllService.RoleService.EnsureBuiltIns(); err != nil {
		t.Fatal(err)
	}
	if err = db.First(legacyAdmin, legacyAdmin.Id).Error; err != nil || legacyAdmin.RoleId == 0 {
		t.Fatalf("legacy administrator was not assigned the built-in role: %v", err)
	}
	var auditor model.Role
	if err = db.Where("code = ?", "auditor").First(&auditor).Error; err != nil {
		t.Fatal(err)
	}
	user := &model.User{Username: "auditor", IsAdmin: boolPtr(false), RoleId: auditor.Id, Status: model.COMMON_STATUS_ENABLE}
	if err = db.Create(user).Error; err != nil {
		t.Fatal(err)
	}
	if !AllService.RoleService.HasPermission(user, "connection-audit") || AllService.RoleService.HasPermission(user, "users") {
		t.Fatal("built-in auditor permissions are incorrect")
	}
	deviceViewer := &model.Role{Name: "只看设备", Code: "device-viewer", Status: model.COMMON_STATUS_ENABLE}
	if err = AllService.RoleService.Create(deviceViewer, []string{"devices"}); err != nil {
		t.Fatal(err)
	}
	if AllService.RoleService.CanAssignRole(user, deviceViewer.Id) || !AllService.RoleService.CanAssignRole(legacyAdmin, deviceViewer.Id) {
		t.Fatal("role assignment allowed privilege escalation or rejected an administrator")
	}
	assigned := &model.User{Username: "viewer", IsAdmin: boolPtr(false), RoleId: deviceViewer.Id, Status: model.COMMON_STATUS_ENABLE}
	if err = db.Create(assigned).Error; err != nil {
		t.Fatal(err)
	}
	if err = AllService.RoleService.Delete(deviceViewer.Id); err == nil {
		t.Fatal("assigned role was deleted")
	}
	if err = AllService.RoleService.Create(&model.Role{Name: "非法", Code: "invalid", Status: model.COMMON_STATUS_ENABLE}, []string{"unknown"}); err == nil {
		t.Fatal("unknown menu permission was accepted")
	}
}

func boolPtr(value bool) *bool { return &value }
