package repair

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/modules/auth"
)

// Module 维修记录模块, 负责维修过程录入与完工闭环。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
}

// New 构造维修记录模块, faults 为故障模块提供的端口实现。
func New(db *gorm.DB, faults FaultPort) *Module {
	repository := NewRepository(db)
	service := NewService(repository, faults)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
	}
}

// Repository 暴露仓储, 供状态查询模块装配。
func (m *Module) Repository() *Repository { return m.repository }

// Service 暴露维修服务, 供状态看板复用数据范围聚合。
func (m *Module) Service() *Service { return m.service }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "维修记录" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&Repair{}} }

// RegisterRoutes 实现 module.Module 接口。
// 行级归属(维修人员仅限本人记录)在 service 层依据同一数据范围函数二次强制,
// 因此即便有人绕过页面直接调接口, 结论也与列表/导出/看板一致。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/repairs")
	{
		group.GET("", auth.RequirePermission(auth.PermRepairRead), m.handler.List)
		group.POST("", auth.RequirePermission(auth.PermRepairCreate), m.handler.Create)
		group.GET("/meta", auth.RequirePermission(auth.PermRepairRead), m.handler.Metadata)
		group.GET("/statistics", auth.RequirePermission(auth.PermRepairRead), m.handler.Statistics)
		group.GET("/export", auth.RequirePermission(auth.PermExportRepair), m.handler.Export)
		group.GET("/fault/:faultId", auth.RequirePermission(auth.PermRepairRead), m.handler.ListByFault)
		group.GET("/:id", auth.RequirePermission(auth.PermRepairRead), m.handler.Get)
		group.PUT("/:id", auth.RequirePermission(auth.PermRepairUpdate), m.handler.Update)
		group.POST("/:id/finish", auth.RequirePermission(auth.PermRepairFinish), m.handler.Finish)
		group.POST("/:id/assign", auth.RequirePermission(auth.PermRepairAssign), m.handler.Assign)
		group.DELETE("/:id", auth.RequirePermission(auth.PermRepairDelete), m.handler.Delete)
	}
}
