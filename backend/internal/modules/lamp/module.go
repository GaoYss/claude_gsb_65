package lamp

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/modules/auth"
)

// Module 路灯台账模块, 负责档案维护与运行状态。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
	guard      *auth.Guard
}

// New 构造路灯台账模块。
func New(db *gorm.DB, guard *auth.Guard) *Module {
	repository := NewRepository(db)
	service := NewService(repository)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
		guard:      guard,
	}
}

// Service 暴露模块业务服务, 供其它模块装配使用。
func (m *Module) Service() *Service { return m.service }

// Repository 暴露仓储, 供状态查询模块装配只读视图。
func (m *Module) Repository() *Repository { return m.repository }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "路灯台账" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&Lamp{}} }

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/lamps")
	group.Use(m.guard.Authenticate(true))
	{
		group.GET("", m.guard.Require(auth.PermLampView, "lamp.view", "lamp"), m.handler.List)
		group.POST("", m.guard.Require(auth.PermLampManage, "lamp.create", "lamp"), m.handler.Create)
		group.GET("/options", m.guard.Require(auth.PermLampView, "lamp.view", "lamp"), m.handler.Options)
		group.GET("/statistics", m.guard.Require(auth.PermLampView, "lamp.view", "lamp"), m.handler.Statistics)
		group.GET("/:id", m.guard.Require(auth.PermLampView, "lamp.view", "lamp"), m.handler.Get)
		group.PUT("/:id", m.guard.Require(auth.PermLampManage, "lamp.update", "lamp"), m.handler.Update)
		group.DELETE("/:id", m.guard.Require(auth.PermLampManage, "lamp.delete", "lamp"), m.handler.Delete)
	}
}
