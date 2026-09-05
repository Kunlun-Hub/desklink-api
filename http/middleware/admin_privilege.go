package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

// AdminPrivilege ...
func AdminPrivilege() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)

		permission := permissionForPath(c.Request.URL.Path)
		if service.AllService.RoleService == nil || permission == "" || !service.AllService.RoleService.HasPermission(u, permission) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}
		if strings.Contains(strings.TrimPrefix(c.Request.URL.Path, "/api/admin/"), "/credentials/") && !service.AllService.UserService.IsAdmin(u) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}
		if service.AllService.RoleService.IsReadOnly(u) && c.Request.Method != http.MethodGet {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// permissionForPath keeps route declarations readable while making every
// protected admin endpoint use the same permission contract as the sidebar.
func permissionForPath(path string) string {
	path = strings.TrimPrefix(path, "/api/admin/")
	first := strings.Split(path, "/")[0]
	switch first {
	case "peer":
		return "devices"
	case "user":
		return "users"
	case "group":
		return "groups"
	case "device_group":
		return "device-groups"
	case "address_book":
		return "address-books"
	case "address_book_collection":
		return "collections"
	case "address_book_collection_rule":
		return "collection-rules"
	case "tag":
		return "tags"
	case "login_log":
		return "login-logs"
	case "audit_conn":
		return "connection-audit"
	case "audit_file":
		return "file-audit"
	case "access-rules":
		return "access-rules"
	case "recordings":
		return "recordings"
	case "user_token":
		return "tokens"
	case "share_record":
		return "shares"
	case "rustdesk":
		return "commands"
	case "oauth":
		return "oauth"
	case "config":
		return "settings"
	default:
		return ""
	}
}

func RolePrivilege() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := service.AllService.UserService.CurUser(c)
		if !service.AllService.UserService.IsAdmin(u) {
			response.Fail(c, 403, response.TranslateMsg(c, "NoAccess"))
			c.Abort()
			return
		}
		c.Next()
	}
}
