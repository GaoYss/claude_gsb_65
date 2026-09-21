package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 认证与权限模块: 登录会话、用户角色管理、操作审计。
type Module struct {
	users   *UserRepository
	audits  *AuditRepository
	tokens  *TokenService
	service *Service
	guard   *Guard
	handler *Handler
}

// New 构造认证模块。secret 为令牌签名密钥, ttl 为令牌有效期。
func New(db *gorm.DB, secret string, ttl time.Duration) *Module {
	users := NewUserRepository(db)
	audits := NewAuditRepository(db)
	tokens := NewTokenService(secret, ttl)
	service := NewService(users, tokens)
	guard := NewGuard(users, tokens, NewAuditService(audits))
	handler := NewHandler(service, guard)
	return &Module{
		users:   users,
		audits:  audits,
		tokens:  tokens,
		service: service,
		guard:   guard,
		handler: handler,
	}
}

// Guard 暴露统一鉴权守卫, 供其它业务模块挂载路由时复用同一个判定入口。
func (m *Module) Guard() *Guard { return m.guard }

// Service 暴露认证服务。
func (m *Module) Service() *Service { return m.service }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "认证与权限" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&User{}, &AuditLog{}} }

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	// 公开接口: 登录与权限目录无需登录。
	public := api.Group("/auth")
	public.POST("/login", m.guard.Authenticate(false), m.handler.Login)
	public.GET("/catalog", m.guard.Authenticate(false), m.handler.Catalog)

	// 已登录即可访问的会话接口。
	session := api.Group("/auth")
	session.Use(m.guard.Authenticate(true))
	session.POST("/logout", m.handler.Logout)
	session.GET("/profile", m.handler.Profile)
	session.PUT("/password", m.handler.ChangePassword)

	// 用户与角色管理: 仅管理岗。
	users := api.Group("/users")
	users.Use(m.guard.Authenticate(true), m.guard.Require(PermUserManage, "user.manage", "user"))
	{
		users.GET("", m.handler.ListUsers)
		users.POST("", m.handler.CreateUser)
		users.PUT("/:id", m.handler.UpdateUser)
		users.POST("/:id/reset-password", m.handler.ResetPassword)
	}

	// 操作审计: 仅管理岗可查, 只读。
	audits := api.Group("/audit-logs")
	audits.Use(m.guard.Authenticate(true), m.guard.Require(PermAuditView, "audit.view", "audit"))
	{
		audits.GET("", m.handler.ListAudit)
	}
}
