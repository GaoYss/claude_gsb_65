package auth

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=128"`
}

// UserListQuery 用户列表查询参数。
type UserListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Role     string `form:"role"`
	Active   *bool  `form:"active"`
}

// AuditListQuery 审计日志查询参数。
type AuditListQuery struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
	ActorUsername string `form:"actor_username"`
	Action        string `form:"action"`
	Result        string `form:"result"`
	ResourceType  string `form:"resource_type"`
	Keyword       string `form:"keyword"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
}

// MeResponse 当前登录用户与其权限点。
type MeResponse struct {
	Profile *Profile     `json:"profile"`
	Roles   []RoleOption `json:"roles"`
}

// RoleOption 角色选项。
type RoleOption struct {
	Value       string   `json:"value"`
	Label       string   `json:"label"`
	Permissions []string `json:"permissions"`
}
