package auth

import "time"

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

// UserProfile 用户信息。
type UserProfile struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	RoleLabel   string `json:"role_label"`
	Active      bool   `json:"active"`
}

// LoginResult 登录/会话信息, permissions 为该用户实际拥有的权限点列表。
type LoginResult struct {
	Token       string      `json:"token,omitempty"`
	ExpiresAt   time.Time   `json:"expires_at,omitempty"`
	User        UserProfile `json:"user"`
	Permissions []string    `json:"permissions"`
	Roles       []RoleMeta  `json:"roles"`
}

// UserUpsertRequest 新建用户请求。
type UserUpsertRequest struct {
	Username    string `json:"username" binding:"required,min=3,max=64"`
	Password    string `json:"password" binding:"required,min=6,max=128"`
	DisplayName string `json:"display_name" binding:"max=64"`
	Role        string `json:"role" binding:"required,oneof=registrar repair admin"`
}

// UserUpdateRequest 修改用户请求, 仅更新显式提交的字段。
type UserUpdateRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,max=64"`
	Role        *string `json:"role" binding:"omitempty,oneof=registrar repair admin"`
	Active      *bool   `json:"active"`
}

// PasswordResetRequest 管理岗重置密码。
type PasswordResetRequest struct {
	Password string `json:"password" binding:"required,min=6,max=128"`
}

// ChangePasswordRequest 用户修改自身密码。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,max=128"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

// AuditQuery 审计日志查询条件。
type AuditQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Actor    string `form:"actor"`
	Action   string `form:"action"`
	Resource string `form:"resource"`
	Result   string `form:"result"`
	Keyword  string `form:"keyword"`
	From     string `form:"from"`
	To       string `form:"to"`
}

// PermissionCatalogResponse 权限目录响应。
type PermissionCatalogResponse struct {
	Roles       []RoleMeta       `json:"roles"`
	Permissions []PermissionMeta `json:"permissions"`
}
