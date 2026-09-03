package api

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Recording struct{}

type recordingCompleteForm struct {
	DurationMs  int64                           `json:"duration_ms"`
	Sha256      string                          `json:"sha256"`
	CursorTrack []service.RecordingCursorSample `json:"cursor_track"`
}

func (r *Recording) Policy(c *gin.Context) {
	peerId := strings.TrimSpace(c.Query("peer_id"))
	uuid := strings.TrimSpace(c.Query("uuid"))
	if peerId == "" || len(peerId) > 64 || uuid == "" {
		response.Fail(c, 101, "peer_id is required")
		return
	}
	peer := service.AllService.PeerService.FindById(peerId)
	if peer.RowId == 0 || peer.Uuid != uuid {
		response.Fail(c, 101, "device identity does not match")
		return
	}
	enabled, err := service.AllService.RecordingService.IsEnabled(peerId)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, gin.H{
		"enabled":    enabled,
		"forced":     enabled,
		"chunk_size": service.AllService.RecordingService.MaxChunkSize(),
	})
}

func (r *Recording) Init(c *gin.Context) {
	form := &service.RecordingInit{}
	if err := c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	recording, token, err := service.AllService.RecordingService.InitUpload(form)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, gin.H{
		"upload_id":    recording.UploadId,
		"upload_token": token,
		"chunk_size":   service.AllService.RecordingService.MaxChunkSize(),
	})
}

func uploadToken(c *gin.Context) string {
	token := strings.TrimSpace(c.GetHeader("X-Upload-Token"))
	if token == "" {
		token = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	}
	return token
}

func (r *Recording) Chunk(c *gin.Context) {
	recording, err := service.AllService.RecordingService.Authorized(c.Param("upload_id"), uploadToken(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	offset, err := strconv.ParseInt(c.Query("offset"), 10, 64)
	if err != nil {
		response.Error(c, "invalid offset")
		return
	}
	if c.Request.ContentLength < 0 {
		response.Error(c, "Content-Length is required")
		return
	}
	written, err := service.AllService.RecordingService.WriteChunk(recording, offset, c.Request.Body, c.Request.ContentLength)
	if err != nil {
		response.Error(c, err.Error())
		return
	}
	response.Success(c, gin.H{"written": written, "next_offset": offset + written})
}

func (r *Recording) Complete(c *gin.Context) {
	recording, err := service.AllService.RecordingService.Authorized(c.Param("upload_id"), uploadToken(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	form := &recordingCompleteForm{}
	if err = c.ShouldBindJSON(form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if err = service.AllService.RecordingService.Complete(recording, form.DurationMs, form.Sha256, form.CursorTrack); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Recording) Cancel(c *gin.Context) {
	recording, err := service.AllService.RecordingService.Authorized(c.Param("upload_id"), uploadToken(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if err = service.AllService.RecordingService.Cancel(recording); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Recording) Content(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	download := c.Query("download") == "1"
	cursor := download && c.Query("cursor") == "1"
	if err != nil || !service.VerifyRecordingAccessToken(c.Query("token"), uint(id), download, cursor) {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	recording, err := service.AllService.RecordingService.Info(uint(id))
	if err != nil || (recording.Status != "complete" && !(c.Query("download") == "1" && recording.Size > 0)) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if download {
		name := strings.ReplaceAll(recording.OriginalName, "\"", "")
		if cursor {
			name = strings.TrimSuffix(name, filepath.Ext(name)) + "-with-cursor.mp4"
		}
		c.Header("Content-Disposition", "attachment; filename=\""+name+"\"")
	}
	contentType := map[string]string{"webm": "video/webm", "mp4": "video/mp4"}[recording.Container]
	preview := !download && recording.PreviewStorageName != ""
	if preview {
		contentType = "video/mp4"
	}
	var path string
	var cleanup func()
	if cursor {
		path, cleanup, err = service.AllService.RecordingService.MaterializeCursorRecording(recording)
		contentType = "video/mp4"
	} else {
		path, cleanup, err = service.AllService.RecordingService.MaterializeRecordingObject(recording, preview)
	}
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer cleanup()
	c.Header("Content-Type", contentType)
	c.File(path)
}
