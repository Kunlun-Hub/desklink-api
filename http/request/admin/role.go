package admin

import "github.com/lejianwen/rustdesk-api/v2/model"

type RoleForm struct {
	Id          uint             `json:"id"`
	Name        string           `json:"name" validate:"required,gte=2,lte=64"`
	Code        string           `json:"code" validate:"required,gte=2,lte=64"`
	Status      model.StatusCode `json:"status" validate:"required,gte=1,lte=2"`
	Permissions []string         `json:"permissions"`
	Remark      string           `json:"remark"`
}

func (f *RoleForm) ToRole() *model.Role {
	return &model.Role{Id: f.Id, Name: f.Name, Code: f.Code, Status: f.Status, Remark: f.Remark}
}

type RoleQuery struct {
	PageQuery
	Name string `form:"name"`
}

type RoleDeleteForm struct {
	Id uint `json:"id" validate:"required"`
}
