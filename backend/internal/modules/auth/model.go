package auth

import "time"

// User 系统登录账号, 角色决定其可操作范围。
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:128;not null" json:"-"`
	DisplayName  string    `gorm:"size:64;not null" json:"display_name"`
	Role         string    `gorm:"size:32;index;not null" json:"role"`
	Active       bool      `gorm:"index;not null;default:true" json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (User) TableName() string { return "sys_user" }

// 审计判定结果。
const (
	AuditResultAllowed = "allowed"
	AuditResultDenied  = "denied"
)

// AuditLog 操作审计记录。
//
// 该表为只追加(append-only)表: 应用层不提供任何更新/删除方法,
// 数据库层另有触发器拒绝 UPDATE/DELETE。每条记录冗余操作者当时的角色与
// 权限快照, 因此之后调整角色权限既不会隐藏也不会改写历史操作记录。
type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `gorm:"index;not null" json:"created_at"`
	OccurredAt time.Time `gorm:"index;not null" json:"occurred_at"`

	// 操作者信息(未登录访问时允许为空)。
	ActorID       uint   `gorm:"index" json:"actor_id"`
	ActorUsername string `gorm:"size:64;index" json:"actor_username"`
	ActorName     string `gorm:"size:64" json:"actor_name"`
	ActorRole     string `gorm:"size:32;index" json:"actor_role"`
	RoleSnapshot  string `gorm:"size:32;not null" json:"role_snapshot"`
	PermSnapshot  string `gorm:"size:1024;not null" json:"perm_snapshot"` // 操作发生时持有的权限点(逗号分隔)

	// 操作目标。
	Action      string `gorm:"size:64;index;not null" json:"action"`   // 如 fault.create / repair.finish
	Resource    string `gorm:"size:32;index;not null" json:"resource"` // lamp / fault / repair / export / auth / user
	ResourceID  string `gorm:"size:64;index" json:"resource_id"`       // 资源标识(单号或主键)
	Description string `gorm:"size:255" json:"description"`            // 人类可读描述

	// 请求快照。
	Method      string `gorm:"size:16" json:"method"`
	Path        string `gorm:"size:255" json:"path"`
	RequestHash string `gorm:"size:64" json:"request_hash"` // method+path+资源主键+参数的稳定哈希, 便于定位同一请求
	IP          string `gorm:"size:64" json:"ip"`
	UserAgent   string `gorm:"size:255" json:"user_agent"`
	RequestID   string `gorm:"size:64;index" json:"request_id"`

	// 判定结果。
	Result             string `gorm:"size:16;index;not null" json:"result"` // allowed / denied
	RequiredPermission string `gorm:"size:64" json:"required_permission"`   // 被拒时缺失的授权点
	Reason             string `gorm:"size:255" json:"reason"`               // 拒绝原因 / 补充说明
	HTTPStatus         int    `json:"http_status"`
}

// TableName 指定表名。
func (AuditLog) TableName() string { return "audit_log" }
