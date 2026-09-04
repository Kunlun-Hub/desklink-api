package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type AccessRule struct{}

func (a *AccessRule) List(c *gin.Context) {
	page, _ := strconv.ParseUint(c.DefaultQuery("page", "1"), 10, 32)
	size, _ := strconv.ParseUint(c.DefaultQuery("page_size", "20"), 10, 32)
	result, err := service.AllService.ListAccessRules(uint(page), uint(size))
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, result)
}

func (a *AccessRule) Save(c *gin.Context) {
	var rule model.AccessRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if err := service.AllService.SaveAccessRule(&rule); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, rule)
}

func (a *AccessRule) Delete(c *gin.Context) {
	var form struct {
		Id uint `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&form); err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	if err := service.DB.Delete(&model.AccessRule{}, form.Id).Error; err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}
