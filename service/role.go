package service

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/model/custom_types"
	"gorm.io/gorm"
)

type RoleService struct{}

var builtInRoles = []struct {
	code        string
	name        string
	permissions []string
}{
	{code: "admin", name: "管理员", permissions: []string{"*"}},
	{code: "auditor", name: "审计员", permissions: []string{"login-logs", "connection-audit", "file-audit", "recordings", "shares"}},
	{code: "operator", name: "操作员", permissions: []string{"devices", "address-books", "collections", "collection-rules", "tags", "commands"}},
}

func encodeRolePermissions(values []string) custom_types.AutoJson {
	if values == nil {
		values = []string{}
	}
	sort.Strings(values)
	b, _ := json.Marshal(values)
	return custom_types.AutoJson(b)
}

func validateRolePermissions(values []string) error {
	allowed := make(map[string]struct{}, len(model.ConsolePermissions))
	for _, permission := range model.ConsolePermissions {
		allowed[permission.Key] = struct{}{}
	}
	for _, permission := range values {
		if permission == "*" {
			return errors.New("wildcard permission is reserved for the administrator")
		}
		if _, ok := allowed[permission]; !ok {
			return errors.New("unknown menu permission: " + permission)
		}
	}
	return nil
}

func (rs *RoleService) EnsureBuiltIns() error {
	for _, builtin := range builtInRoles {
		var role model.Role
		err := DB.Where("code = ?", builtin.code).First(&role).Error
		permissions := encodeRolePermissions(builtin.permissions)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err = DB.Create(&model.Role{Name: builtin.name, Code: builtin.code, BuiltIn: true, Status: model.COMMON_STATUS_ENABLE, Permissions: permissions}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		updates := map[string]interface{}{"name": builtin.name, "built_in": true, "status": model.COMMON_STATUS_ENABLE}
		if len(role.Permissions) == 0 || string(role.Permissions) == "null" {
			updates["permissions"] = permissions
		}
		if err = DB.Model(&role).Updates(updates).Error; err != nil {
			return err
		}
	}
	var adminRole model.Role
	if err := DB.Where("code = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}
	return DB.Model(&model.User{}).Where("is_admin = ? AND (role_id = 0 OR role_id IS NULL)", true).Update("role_id", adminRole.Id).Error
}

func (rs *RoleService) Permissions(role *model.Role) []string {
	if role == nil || role.Status != model.COMMON_STATUS_ENABLE {
		return nil
	}
	var permissions []string
	if err := json.Unmarshal(role.Permissions, &permissions); err != nil {
		return nil
	}
	return permissions
}

func (rs *RoleService) RoleForUser(user *model.User) *model.Role {
	if user == nil || user.RoleId == 0 {
		return nil
	}
	var role model.Role
	if DB.Where("id = ?", user.RoleId).First(&role).Error != nil {
		return nil
	}
	return &role
}

func (rs *RoleService) PermissionsForUser(user *model.User) []string {
	if user == nil {
		return nil
	}
	if user.IsAdmin != nil && *user.IsAdmin {
		return []string{"*"}
	}
	return rs.Permissions(rs.RoleForUser(user))
}

func (rs *RoleService) HasPermission(user *model.User, permission string) bool {
	for _, value := range rs.PermissionsForUser(user) {
		if value == "*" || value == permission {
			return true
		}
	}
	return false
}

func (rs *RoleService) IsReadOnly(user *model.User) bool {
	role := rs.RoleForUser(user)
	return role != nil && role.Code == "auditor"
}

func (rs *RoleService) CanAssignRole(user *model.User, roleId uint) bool {
	if roleId == 0 || (AllService != nil && AllService.UserService != nil && AllService.UserService.IsAdmin(user)) {
		return true
	}
	target := rs.InfoById(roleId)
	if target.Id == 0 || target.Status != model.COMMON_STATUS_ENABLE || target.Code == "admin" {
		return false
	}
	for _, permission := range rs.Permissions(target) {
		if !rs.HasPermission(user, permission) {
			return false
		}
	}
	return true
}

func (rs *RoleService) List(page, pageSize uint, where func(tx *gorm.DB)) (res *model.RoleList) {
	res = &model.RoleList{Pagination: model.Pagination{Page: int64(page), PageSize: int64(pageSize)}}
	tx := DB.Model(&model.Role{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize)).Order("built_in desc, id asc").Find(&res.Roles)
	return
}

func (rs *RoleService) InfoById(id uint) *model.Role {
	role := &model.Role{}
	DB.First(role, id)
	return role
}

func (rs *RoleService) Create(role *model.Role, permissions []string) error {
	if role.Name == "" || role.Code == "" {
		return errors.New("role name and code are required")
	}
	if err := validateRolePermissions(permissions); err != nil {
		return err
	}
	role.BuiltIn = false
	role.Permissions = encodeRolePermissions(permissions)
	return DB.Create(role).Error
}

func (rs *RoleService) Update(role *model.Role, permissions []string) error {
	current := rs.InfoById(role.Id)
	if current.Id == 0 {
		return gorm.ErrRecordNotFound
	}
	if current.BuiltIn {
		return errors.New("built-in roles cannot be modified")
	}
	if err := validateRolePermissions(permissions); err != nil {
		return err
	}
	role.BuiltIn = false
	role.Permissions = encodeRolePermissions(permissions)
	return DB.Model(current).Updates(map[string]interface{}{"name": role.Name, "code": role.Code, "status": role.Status, "permissions": role.Permissions, "remark": role.Remark}).Error
}

func (rs *RoleService) Delete(id uint) error {
	role := rs.InfoById(id)
	if role.Id == 0 {
		return gorm.ErrRecordNotFound
	}
	if role.BuiltIn {
		return errors.New("built-in roles cannot be deleted")
	}
	var count int64
	DB.Model(&model.User{}).Where("role_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("role is still assigned to users")
	}
	return DB.Delete(role).Error
}
