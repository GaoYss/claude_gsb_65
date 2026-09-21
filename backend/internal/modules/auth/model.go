package auth

import "time"

// 系统内置角色。
const (
	// RoleRegistrar 登记人员: 只做故障登记与查询。
	RoleRegistrar = "registrar"
	// RoleRepairman 维修人员: 只推进分配给自己的维修记录。
	RoleRepairman = "repairman"
	// RoleManager 管理岗: 全局可见, 负责改派、例外流转、台账维护、用户与授权管理。
	RoleManager = "manager"
)

// Roles 返回全部内置角色。
func Roles() []string {
	return []string{RoleRegistrar, RoleRepairman, RoleManager}
}

// RoleLabel 返回角色的中文名称。
func RoleLabel(role string) string {
	switch role {
	case RoleRegistrar:
		return "登记人员"
	case RoleRepairman:
		return "维修人员"
	case RoleManager:
		return "管理岗"
	default:
		return role
	}
}

// IsValidRole 校验角色取值。
func IsValidRole(role string) bool {
	for _, item := range Roles() {
		if item == role {
			return true
		}
	}
	return false
}

// 审计结果取值。
const (
	AuditResultAllowed = "allowed" // 操作已放行(成功或业务失败均说明通过了授权)
	AuditResultDenied  = "denied"  // 操作因越权被拒绝
)

// User 系统用户, 用户名用于登录, 角色决定权限集合。
// 用户只做停用(active=false)不做物理删除, 以保证历史审计记录始终可追溯。
type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"size:128;not null" json:"-"`
	DisplayName  string     `gorm:"size:64;not null" json:"display_name"`
	Role         string     `gorm:"size:32;index;not null" json:"role"`
	Team         string     `gorm:"size:64" json:"team"`
	Active       bool       `gorm:"index;not null;default:true" json:"active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (User) TableName() string { return "sys_user" }

// Session 登录会话, 只保存登录令牌的 SHA-256 摘要, 不落原始令牌。
type Session struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名。
func (Session) TableName() string { return "sys_session" }

// AuditLog 操作审计记录。
// 该表只增不改: 仓储层不提供更新与删除方法, 权限变更也无法隐藏或改写历史记录。
// 操作人信息(用户名/角色)冗余落库, 即使用户事后被停用或调整角色, 历史记录仍保持当时原貌。
type AuditLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	CreatedAt         time.Time `gorm:"index;not null" json:"created_at"`
	ActorID           uint      `gorm:"index" json:"actor_id"`
	ActorUsername     string    `gorm:"size:64;index" json:"actor_username"`
	ActorRole         string    `gorm:"size:32;index" json:"actor_role"`
	Action            string    `gorm:"size:64;index;not null" json:"action"`
	ResourceType      string    `gorm:"size:32;index" json:"resource_type"`
	ResourceID        uint      `gorm:"index" json:"resource_id"`
	ResourceNo        string    `gorm:"size:64;index" json:"resource_no"`
	Result            string    `gorm:"size:16;index;not null" json:"result"`
	MissingPermission string    `gorm:"size:64" json:"missing_permission,omitempty"`
	Method            string    `gorm:"size:8" json:"method"`
	Path              string    `gorm:"size:255" json:"path"`
	RequestID         string    `gorm:"size:64;index" json:"request_id"`
	ClientIP          string    `gorm:"size:64" json:"client_ip"`
	StatusCode        int       `json:"status_code"`
	Detail            string    `gorm:"size:1024" json:"detail,omitempty"`
}

// TableName 指定表名。
func (AuditLog) TableName() string { return "sys_audit_log" }
