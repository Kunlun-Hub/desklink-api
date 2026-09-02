package api

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"net/http"
	"strings"
	"time"
)

type Index struct {
}

func (i *Index) SwitchGrant(c *gin.Context) {
	form := &requstform.SwitchGrantForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"accepted": false, "error": "invalid request"})
		return
	}
	internalURL := strings.TrimRight(global.Config.Rustdesk.HbbsInternalUrl, "/")
	if internalURL == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"accepted": false, "error": "hbbs switch grants are not configured"})
		return
	}
	body, err := json.Marshal(form)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"accepted": false})
		return
	}
	request, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, internalURL+"/switch-grant", bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"accepted": false})
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if key := global.Config.Rustdesk.HbbsInternalKey; key != "" {
		request.Header.Set("X-DeskLink-Internal-Key", key)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"accepted": false, "error": "hbbs unavailable"})
		return
	}
	defer response.Body.Close()
	var result gin.H
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"accepted": false, "error": "invalid hbbs response"})
		return
	}
	c.JSON(response.StatusCode, result)
}

// Index 首页
// @Tags 首页
// @Summary 首页
// @Description 首页
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router / [get]
func (i *Index) Index(c *gin.Context) {
	response.Success(
		c,
		"DeskLink Community API",
	)
}

// Heartbeat 心跳
// @Tags 首页
// @Summary 心跳
// @Description 心跳
// @Accept  json
// @Produce  json
// @Success 200 {object} nil
// @Failure 500 {object} response.Response
// @Router /heartbeat [post]
func (i *Index) Heartbeat(c *gin.Context) {
	info := &requstform.PeerInfoInHeartbeat{}
	err := c.ShouldBindJSON(info)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	if info.Uuid == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	peer := service.AllService.PeerService.FindById(info.Id)
	if peer == nil || peer.RowId == 0 || peer.Uuid == "" || peer.Uuid != info.Uuid {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	//如果在40s以内则不更新
	if time.Now().Unix()-peer.LastOnlineTime >= 30 {
		upp := &model.Peer{RowId: peer.RowId, LastOnlineTime: time.Now().Unix(), LastOnlineIp: c.ClientIP()}
		service.AllService.PeerService.Update(upp)
	}
	enabled, policyErr := service.AllService.RecordingService.IsEnabled(info.Id)
	if policyErr != nil {
		global.Logger.Warnf("recording policy lookup failed for %s: %v", info.Id, policyErr)
	}
	c.JSON(http.StatusOK, gin.H{
		"recording": gin.H{
			"enabled":    enabled,
			"forced":     enabled,
			"chunk_size": service.AllService.RecordingService.MaxChunkSize(),
		},
	})
}

// Version 版本
// @Tags 首页
// @Summary 版本
// @Description 版本
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /version [get]
func (i *Index) Version(c *gin.Context) {
	//读取resources/version文件
	v := service.AllService.AppService.GetAppVersion()
	response.Success(
		c,
		v,
	)
}
