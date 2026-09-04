package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type accessCheckForm struct {
	TargetPeerID string `json:"target_peer_id"`
	SourceIP     string `json:"source_ip"`
	SourcePeerID string `json:"source_peer_id"`
	Token        string `json:"token"`
}

func (i *Index) AccessCheck(c *gin.Context) {
	configured := global.Config.Rustdesk.HbbsInternalKey
	provided := c.GetHeader("X-DeskLink-Internal-Key")
	if configured == "" || subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"allowed": false})
		return
	}
	var form accessCheckForm
	if err := c.ShouldBindJSON(&form); err != nil || strings.TrimSpace(form.TargetPeerID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"allowed": false})
		return
	}
	var userID uint
	trustedSourcePeerID := ""
	if form.Token != "" {
		if user, userToken := service.AllService.UserService.InfoByAccessToken(form.Token); user != nil {
			userID = user.Id
			if userToken != nil {
				trustedSourcePeerID = strings.TrimSpace(userToken.DeviceId)
			}
		}
	}
	allowed := service.AllService.CheckAccessRule(service.AccessCheck{
		TargetPeerID: strings.TrimSpace(form.TargetPeerID), SourceIP: strings.TrimSpace(form.SourceIP),
		SourcePeerID: trustedSourcePeerID, SourceUserID: userID,
	})
	c.JSON(http.StatusOK, gin.H{"allowed": allowed, "user_id": userID})
}
