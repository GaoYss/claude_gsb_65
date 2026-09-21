package auth

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 认证与授权模块: 用户、会话、权限策略与审计日志。
type Module struct {
	service *Service
	handler *Handler
}

// New 构造认证模块。
func New(db *gorm.DB) *Module {
	service := NewService(db, DefaultSessionTTL)
	return &Module{service: service, handler: NewHandler(service)}
}

// NewWithTTL 使用指定会话有效期构造模块(主要用于测试)。
func NewWithTTL(db *gorm.DB, ttl time.Duration) *Module {
	service := NewService(db, ttl)
	return &Module{service: service, handler: NewHandler(service)}
}

// Service 暴露认证服务, 供 bootstrap 装配全局中间件。
func (m *Module) Service() *Service { return m.service }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "认证授权" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any {
	return []any{&User{}, &Session{}, &AuditLog{}}
}

// RegisterRoutes 实现 module.Module 接口。
// 注意: Authenticate / Audit 作为全局中间件由 bootstrap 挂载,
// 这里只注册 /auth 下的具体路由及其权限要求。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/auth")
	{
		group.POST("/login", m.handler.Login)

		authed := group.Group("")
		authed.Use(RequireAuth())
		authed.GET("/me", m.handler.Me)
		authed.POST("/logout", m.handler.Logout)

		users := group.Group("/users")
		users.Use(RequirePermission(PermUserManage))
		{
			users.GET("", m.handler.ListUsers)
			users.POST("", m.handler.CreateUser)
			users.PUT("/:id", m.handler.UpdateUser)
		}

		audits := group.Group("/audit-logs")
		audits.Use(RequirePermission(PermAuditRead))
		{
			audits.GET("", m.handler.ListAudit)
		}

		group.GET("/permissions", RequirePermission(PermAuditRead), m.handler.PermissionMeta)
	}
}
