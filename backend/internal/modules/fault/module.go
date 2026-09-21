package fault

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/modules/auth"
)

// Module 故障登记模块, 负责故障上报受理与状态流转。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
	guard      *auth.Guard
}

// New 构造故障登记模块, lamps 为路灯台账模块提供的端口实现。
func New(db *gorm.DB, lamps LampPort, guard *auth.Guard) *Module {
	repository := NewRepository(db)
	service := NewService(repository, lamps)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
		guard:      guard,
	}
}

// Service 暴露业务服务, 供维修模块装配端口使用。
func (m *Module) Service() *Service { return m.service }

// Repository 暴露仓储, 供路灯模块装配未闭环故障计数器。
func (m *Module) Repository() *Repository { return m.repository }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "故障登记" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&Fault{}} }

// RegisterRoutes 实现 module.Module 接口。
// 每个路由都经过同一个 Guard 判定, 与页面、导出、看板的结论完全一致。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/faults")
	group.Use(m.guard.Authenticate(true))
	{
		group.GET("", m.guard.Require(auth.PermFaultView, "fault.view", "fault"), m.handler.List)
		group.POST("", m.guard.Require(auth.PermFaultRegister, "fault.register", "fault"), m.handler.Create)
		group.GET("/meta", m.guard.Require(auth.PermFaultView, "fault.view", "fault"), m.handler.Metadata)
		group.GET("/:id", m.guard.Require(auth.PermFaultView, "fault.view", "fault"), m.handler.Get)
		group.PUT("/:id", m.guard.Require(auth.PermFaultUpdate, "fault.update", "fault"), m.handler.Update)
		group.DELETE("/:id", m.guard.Require(auth.PermFaultDelete, "fault.delete", "fault"), m.handler.Delete)
		group.POST("/:id/close", m.guard.Require(auth.PermFaultClose, "fault.close", "fault"), m.handler.Close)
		group.POST("/:id/reassign", m.guard.Require(auth.PermFaultReassign, "fault.reassign", "fault"), m.handler.Reassign)
		group.POST("/:id/exception", m.guard.Require(auth.PermFaultException, "fault.exception", "fault"), m.handler.Exception)
	}
}
