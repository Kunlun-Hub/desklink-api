package admin

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Recording struct{}

type recordingPolicyForm struct {
	Mode          string   `json:"mode" binding:"required"`
	RetentionDays int      `json:"retention_days" binding:"required"`
	PeerIds       []string `json:"peer_ids"`
}

func (r *Recording) Policy(c *gin.Context) {
	policy, peerIds, err := service.AllService.RecordingService.Policy()
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, gin.H{"mode": policy.Mode, "retention_days": policy.RetentionDays, "peer_ids": peerIds})
}

func (r *Recording) SavePolicy(c *gin.Context) {
	form := &recordingPolicyForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if err := service.AllService.RecordingService.SavePolicy(form.Mode, form.RetentionDays, form.PeerIds); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	r.Policy(c)
}

func (r *Recording) Storage(c *gin.Context) {
	config, err := service.AllService.RecordingService.StorageConfig()
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, config)
}

func (r *Recording) SaveStorage(c *gin.Context) {
	form := &service.RecordingStorageConfig{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	config, err := service.AllService.RecordingService.SaveStorageConfig(*form)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, config)
}

func (r *Recording) List(c *gin.Context) {
	page, _ := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	pageSize, _ := strconv.ParseUint(c.DefaultQuery("page_size", "20"), 10, 32)
	if pageSize == 0 || pageSize > 100 {
		pageSize = 20
	}
	startedAfter, _ := strconv.ParseInt(c.Query("started_after"), 10, 64)
	startedBefore, _ := strconv.ParseInt(c.Query("started_before"), 10, 64)
	response.Success(c, service.AllService.RecordingService.List(
		uint(page), uint(pageSize), strings.TrimSpace(c.Query("peer_id")),
		strings.TrimSpace(c.Query("from_peer")), strings.TrimSpace(c.Query("status")),
		startedAfter, startedBefore,
	))
}

func (r *Recording) Access(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, 101, "invalid recording id")
		return
	}
	download := c.Query("download") == "1"
	cursor := download && c.Query("cursor") == "1"
	recording, err := service.AllService.RecordingService.Info(uint(id))
	if err != nil {
		response.Fail(c, 101, "recording not found")
		return
	}
	if cursor {
		status, renderErr := service.AllService.RecordingService.StartCursorRender(recording, c.Query("retry") == "1")
		if renderErr != nil {
			response.Fail(c, 101, renderErr.Error())
			return
		}
		if status != "ready" {
			response.Success(c, gin.H{"status": status, "error": recording.CursorRenderError})
			return
		}
	}
	token, expiresAt, err := service.CreateRecordingAccessToken(uint(id), download, cursor)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	url := "/api/recordings/" + strconv.FormatUint(id, 10) + "/content?token=" + token
	if download {
		url += "&download=1"
	}
	if cursor {
		url += "&cursor=1"
	}
	response.Success(c, gin.H{"status": "ready", "url": url, "expires_at": expiresAt})
}

func (r *Recording) CursorTrack(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, 101, "invalid recording id")
		return
	}
	recording, err := service.AllService.RecordingService.Info(uint(id))
	if err != nil {
		response.Fail(c, 101, "recording not found")
		return
	}
	track := json.RawMessage(recording.CursorTrack)
	if len(track) == 0 || !json.Valid(track) {
		track = json.RawMessage("[]")
	}
	response.Success(c, track)
}

func (r *Recording) Delete(c *gin.Context) {
	form := struct {
		Id uint `json:"id" binding:"required"`
	}{}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	recording, err := service.AllService.RecordingService.Info(form.Id)
	if err != nil {
		response.Fail(c, 101, "recording not found")
		return
	}
	if err = service.AllService.RecordingService.Delete(recording); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Recording) BatchDelete(c *gin.Context) {
	form := struct {
		Ids []uint `json:"ids" binding:"required"`
	}{}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if err := service.AllService.RecordingService.DeleteMany(form.Ids); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}
