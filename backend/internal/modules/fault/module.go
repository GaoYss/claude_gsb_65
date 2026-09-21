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
}

// New 构造故障登记模块, lamps 为路灯台账模块提供的端口实现。
func New(db *gorm.DB, lamps LampPort) *Module {
	repository := NewRepository(db)
	service := NewService(repository, lamps)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
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
// 登记人员: 查询 + 登记 + 修改; 管理岗额外拥有关闭/删除/例外流转。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/faults")
	{
		group.GET("", auth.RequirePermission(auth.PermFaultRead), m.handler.List)
		group.POST("", auth.RequirePermission(auth.PermFaultCreate), m.handler.Create)
		group.GET("/meta", auth.RequirePermission(auth.PermFaultRead), m.handler.Metadata)
		group.GET("/export", auth.RequirePermission(auth.PermExportFault), m.handler.Export)
		group.GET("/:id", auth.RequirePermission(auth.PermFaultRead), m.handler.Get)
		group.PUT("/:id", auth.RequirePermission(auth.PermFaultUpdate), m.handler.Update)
		group.DELETE("/:id", auth.RequirePermission(auth.PermFaultDelete), m.handler.Delete)
		group.POST("/:id/close", auth.RequirePermission(auth.PermFaultClose), m.handler.Close)
		group.POST("/:id/transition", auth.RequirePermission(auth.PermFaultBypass), m.handler.Transition)
	}
}
