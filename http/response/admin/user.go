package admin

import "github.com/lejianwen/rustdesk-api/v2/model"

type LoginPayload struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Avatar      string   `json:"avatar"`
	Token       string   `json:"token"`
	RouteNames  []string `json:"route_names"`
	Permissions []string `json:"permissions"`
	RoleId      uint     `json:"role_id"`
	RoleName    string   `json:"role_name"`
	RoleCode    string   `json:"role_code"`
	IsAdmin     *bool    `json:"is_admin"`
	Nickname    string   `json:"nickname"`
}

func (lp *LoginPayload) FromUser(user *model.User) {
	lp.Username = user.Username
	lp.Email = user.Email
	lp.Avatar = user.Avatar
	lp.Nickname = user.Nickname
	lp.RoleId = user.RoleId
	lp.IsAdmin = user.IsAdmin
	if user.Role != nil {
		lp.RoleName = user.Role.Name
		lp.RoleCode = user.Role.Code
	}
}

type UserOauthItem struct {
	Op     string `json:"op"`
	Status int    `json:"status"`
}
