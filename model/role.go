package model

import "github.com/lejianwen/rustdesk-api/v2/model/custom_types"

// Permission identifies a management-console menu and its protected API scope.
type Permission struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

var ConsolePermissions = []Permission{
	{Key: "devices", Label: "设备"},
	{Key: "users", Label: "用户"},
	{Key: "groups", Label: "用户组"},
	{Key: "device-groups", Label: "设备组"},
	{Key: "address-books", Label: "地址簿"},
	{Key: "collections", Label: "地址簿集合"},
	{Key: "collection-rules", Label: "共享规则"},
	{Key: "tags", Label: "标签"},
	{Key: "login-logs", Label: "登录日志"},
	{Key: "connection-audit", Label: "连接审计"},
	{Key: "access-rules", Label: "授权管理"},
	{Key: "file-audit", Label: "文件审计"},
	{Key: "recordings", Label: "会话录像"},
	{Key: "tokens", Label: "访问令牌"},
	{Key: "shares", Label: "分享记录"},
	{Key: "commands", Label: "服务指令"},
	{Key: "oauth", Label: "OAuth / OIDC"},
	{Key: "settings", Label: "采集设置"},
}

type Role struct {
	Id          uint                  `json:"id" gorm:"primaryKey"`
	Name        string                `json:"name" gorm:"size:64;not null"`
	Code        string                `json:"code" gorm:"size:64;not null;uniqueIndex"`
	BuiltIn     bool                  `json:"built_in" gorm:"default:0;not null;index"`
	Status      StatusCode            `json:"status" gorm:"default:1;not null;index"`
	Permissions custom_types.AutoJson `json:"permissions" gorm:"type:text;not null" swaggertype:"array,string"`
	Remark      string                `json:"remark" gorm:"size:255;default:'';not null"`
	TimeModel
}

type RoleList struct {
	Roles []*Role `json:"list"`
	Pagination
}
