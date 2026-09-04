package api

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	requstform "github.com/lejianwen/rustdesk-api/v2/http/request/api"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"net/http"
	"time"
)

type Peer struct {
}

// SysInfo
// @Tags System
// @Summary 提交系统信息
// @Description 提交系统信息
// @Accept  json
// @Produce  json
// @Param body body requstform.PeerForm true "系统信息表单"
// @Success 200 {string} string "SYSINFO_UPDATED,ID_NOT_FOUND"
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo [post]
func (p *Peer) SysInfo(c *gin.Context) {
	f := &requstform.PeerForm{}
	err := c.ShouldBindBodyWith(f, binding.JSON)
	if err != nil {
		response.Error(c, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	fpe := f.ToPeer()
	pe := service.AllService.PeerService.FindById(f.Id)
	if pe.RowId == 0 {
		pe = f.ToPeer()
		pe.UserId = service.AllService.UserService.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		err = service.AllService.PeerService.Create(pe)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	} else {
		if pe.UserId == 0 {
			pe.UserId = service.AllService.UserService.FindLatestUserIdFromLoginLogByUuid(pe.Uuid, pe.Id)
		}
		fpe.RowId = pe.RowId
		fpe.UserId = pe.UserId
		err = service.AllService.PeerService.Update(fpe)
		if err != nil {
			response.Error(c, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}
	//SYSINFO_UPDATED 上传成功
	//ID_NOT_FOUND 下次心跳会上传
	//直接响应文本
	c.String(http.StatusOK, "SYSINFO_UPDATED")
}

// SysInfoVer
// @Tags System
// @Summary 获取系统版本信息
// @Description 获取系统版本信息
// @Accept  json
// @Produce  json
// @Success 200 {string} string ""
// @Failure 500 {object} response.ErrorResponse
// @Router /sysinfo_ver [post]
func (p *Peer) SysInfoVer(c *gin.Context) {
	//读取resources/version文件
	v := service.AllService.AppService.GetAppVersion()
	// 加上启动时间，方便client上传信息
	v = fmt.Sprintf("%s\n%s", v, service.AllService.AppService.GetStartTime())
	c.String(http.StatusOK, v)
}

func (p *Peer) AgentMetrics(c *gin.Context) {
	form := &requstform.AgentMetricsForm{}
	if err := c.ShouldBindJSON(form); err != nil {
		c.String(http.StatusOK, "INVALID")
		return
	}
	peer := service.AllService.PeerService.FindById(form.Id)
	if peer.RowId == 0 || peer.Uuid == "" || peer.Uuid != form.Uuid {
		c.String(http.StatusOK, "ID_NOT_FOUND")
		return
	}
	disks, err := json.Marshal(form.Disks)
	if err != nil {
		c.String(http.StatusOK, "INVALID")
		return
	}
	updated := &model.Peer{RowId: peer.RowId, CpuUsage: clampPercent(form.CpuUsage), MemoryTotal: form.MemoryTotal, MemoryUsed: form.MemoryUsed, MemoryUsage: clampPercent(form.MemoryUsage), DiskUsage: string(disks), DiskReadBps: form.DiskReadBps, DiskWriteBps: form.DiskWriteBps, MetricsAt: form.Timestamp}
	if updated.MetricsAt <= 0 {
		updated.MetricsAt = time.Now().Unix()
	}
	if err := service.AllService.PeerService.Update(updated); err != nil {
		c.String(http.StatusOK, "FAILED")
		return
	}
	c.String(http.StatusOK, "METRICS_UPDATED")
}

// Detail returns the latest system and agent metrics for a device.
func (p *Peer) Detail(c *gin.Context) {
	peer := service.AllService.PeerService.FindById(c.Param("id"))
	if peer.RowId == 0 {
		response.Fail(c, 404, "device not found")
		return
	}
	var disks []requstform.AgentDiskMetric
	if peer.DiskUsage != "" {
		_ = json.Unmarshal([]byte(peer.DiskUsage), &disks)
	}
	response.Success(c, gin.H{"peer": peer, "disks": disks})
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
