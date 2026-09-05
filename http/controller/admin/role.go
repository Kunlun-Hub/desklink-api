package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	adminRequest "github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
)

type Role struct{}

func (ct *Role) List(c *gin.Context) {
	query := &adminRequest.RoleQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	result := service.AllService.RoleService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.Name != "" {
			tx.Where("name like ? OR code like ?", "%"+query.Name+"%", "%"+query.Name+"%")
		}
	})
	response.Success(c, result)
}

func (ct *Role) Options(c *gin.Context) {
	roles := service.AllService.RoleService.List(1, 1000, func(tx *gorm.DB) {
		tx.Where("status = ?", model.COMMON_STATUS_ENABLE)
	})
	response.Success(c, roles.Roles)
}

func (ct *Role) Permissions(c *gin.Context) {
	response.Success(c, model.ConsolePermissions)
}

func (ct *Role) Create(c *gin.Context) {
	f := &adminRequest.RoleForm{}
	if !ct.bind(c, f) {
		return
	}
	role := f.ToRole()
	role.Code = strings.ToLower(strings.TrimSpace(role.Code))
	if err := service.AllService.RoleService.Create(role, f.Permissions); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, role)
}

func (ct *Role) Update(c *gin.Context) {
	f := &adminRequest.RoleForm{}
	if !ct.bind(c, f) || f.Id == 0 {
		return
	}
	role := f.ToRole()
	role.Code = strings.ToLower(strings.TrimSpace(role.Code))
	if err := service.AllService.RoleService.Update(role, f.Permissions); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *Role) Delete(c *gin.Context) {
	f := &adminRequest.RoleDeleteForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if err := service.AllService.RoleService.Delete(f.Id); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *Role) bind(c *gin.Context, f *adminRequest.RoleForm) bool {
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return false
	}
	if errors := global.Validator.ValidStruct(c, f); len(errors) > 0 {
		response.Fail(c, 101, errors[0])
		return false
	}
	return true
}
