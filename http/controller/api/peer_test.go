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
	if err = db.AutoMigrate(&model.Peer{}, &model.AgentMetricSample{}); err != nil {
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

func TestAgentMetricsStoresHistoryAndClearsZeroRates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent-metrics-history-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.Peer{}, &model.AgentMetricSample{}); err != nil {
		t.Fatal(err)
	}
	service.DB = db
	service.Config = &config.Config{}
	service.AllService = &service.Service{PeerService: &service.PeerService{}}
	peer := &model.Peer{Id: "history-peer", Uuid: "history-uuid", DiskReadBps: 123, DiskWriteBps: 456}
	if err = db.Create(peer).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/metrics", (&Peer{}).AgentMetrics)
	req := httptest.NewRequest(http.MethodPost, "/metrics", strings.NewReader(`{"id":"history-peer","uuid":"history-uuid","cpu_model":"Test CPU","cpu_usage":12.5,"memory_usage":34.5,"disk_read_bps":0,"disk_write_bps":0}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || strings.TrimSpace(resp.Body.String()) != "METRICS_UPDATED" {
		t.Fatalf("unexpected response: %d %q", resp.Code, resp.Body.String())
	}
	var stored model.Peer
	if err = db.First(&stored, peer.RowId).Error; err != nil {
		t.Fatal(err)
	}
	if stored.DiskReadBps != 0 || stored.DiskWriteBps != 0 || stored.CpuModel != "Test CPU" {
		t.Fatalf("zero rates were not persisted: %#v", stored)
	}
	samples, err := (&service.PeerService{}).MetricsHistory("history-peer", 0, 0, 10)
	if err != nil || len(samples) != 1 || samples[0].CpuUsage != 12.5 || samples[0].CpuModel != "Test CPU" {
		t.Fatalf("unexpected stored history: %#v, %v", samples, err)
	}
}
