package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviceCredentialsRevealOnlyRequestedKind(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:admin-device-credentials-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Peer{}, &model.DeviceCredential{}); err != nil {
		t.Fatal(err)
	}
	service.DB = db
	service.Config = &config.Config{Jwt: config.Jwt{Key: "admin-device-credential-test-key"}}
	service.AllService = &service.Service{PeerService: &service.PeerService{}}
	global.Config = *service.Config
	peer := &model.Peer{Id: "admin-credential-peer", Uuid: "admin-credential-uuid"}
	if err = db.Create(peer).Error; err != nil {
		t.Fatal(err)
	}
	temporary, permanent := "temporary-secret", "permanent-secret"
	if err = service.UpsertDeviceCredentials(peer.Id, &temporary, &permanent); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/credentials/:id", (&Peer{}).Credentials)

	request := httptest.NewRequest(http.MethodGet, "/credentials/1", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if strings.Contains(response.Body.String(), temporary) || strings.Contains(response.Body.String(), permanent) {
		t.Fatalf("credential status leaked a password: %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/credentials/1?reveal=1&kind=temporary", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), temporary) || strings.Contains(response.Body.String(), permanent) {
		t.Fatalf("credential reveal returned the wrong scope: %s", response.Body.String())
	}
}
