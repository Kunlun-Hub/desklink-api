package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviceCredentialsRequireMatchingDeviceIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device-credentials-api-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Peer{}, &model.DeviceCredential{}); err != nil {
		t.Fatal(err)
	}
	service.DB = db
	service.Config = &config.Config{Jwt: config.Jwt{Key: "device-credential-test-key"}}
	service.AllService = &service.Service{PeerService: &service.PeerService{}}
	if err = db.Create(&model.Peer{Id: "credential-peer", Uuid: "credential-uuid"}).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/credentials", (&Peer{}).DeviceCredentials)

	request := httptest.NewRequest(http.MethodPost, "/credentials", strings.NewReader(`{"id":"credential-peer","uuid":"wrong-uuid","temporary_password":"secret"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if strings.TrimSpace(response.Body.String()) != "ID_NOT_FOUND" {
		t.Fatalf("mismatched device was accepted: %q", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/credentials", strings.NewReader(`{"id":"credential-peer","uuid":"credential-uuid","temporary_password":"temporary-secret"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if strings.TrimSpace(response.Body.String()) != "CREDENTIALS_UPDATED" {
		t.Fatalf("valid credential upload failed: %q", response.Body.String())
	}
	var stored model.DeviceCredential
	if err = db.First(&stored).Error; err != nil || strings.Contains(stored.EncryptedSecret, "temporary-secret") {
		t.Fatalf("credential was not stored safely: %#v, %v", stored, err)
	}
}
