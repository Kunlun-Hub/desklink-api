package api

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

func TestAgentMetricsRejectsOversizedDiskList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent-metrics-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Peer{}); err != nil {
		t.Fatal(err)
	}
	service.DB = db
	service.Config = &config.Config{}
	service.AllService = &service.Service{PeerService: &service.PeerService{}}
	peer := &model.Peer{Id: "metric-peer", Uuid: "metric-uuid"}
	if err = db.Create(peer).Error; err != nil {
		t.Fatal(err)
	}
	global.Config = config.Config{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/metrics", (&Peer{}).AgentMetrics)
	disks := make([]string, 65)
	for i := range disks {
		disks[i] = `{"mount":"/"}`
	}
	body := `{"id":"metric-peer","uuid":"metric-uuid","disks":[` + strings.Join(disks, ",") + `]}`
	req := httptest.NewRequest(http.MethodPost, "/metrics", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || strings.TrimSpace(resp.Body.String()) != "INVALID" {
		t.Fatalf("unexpected response: %d %q", resp.Code, resp.Body.String())
	}
}
