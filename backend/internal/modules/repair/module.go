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
	guard      *auth.Guard
}

// New 构造维修记录模块, faults 为故障模块提供的端口实现。
func New(db *gorm.DB, faults FaultPort, guard *auth.Guard) *Module {
	repository := NewRepository(db)
	service := NewService(repository, faults)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service, guard),
		guard:      guard,
	}
}

// Repository 暴露仓储, 供状态查询模块装配。
func (m *Module) Repository() *Repository { return m.repository }

// Service 暴露维修服务, 供 bootstrap 注入用户目录端口。
func (m *Module) Service() *Service { return m.service }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "维修记录" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any { return []any{&Repair{}} }

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/repairs")
	group.Use(m.guard.Authenticate(true))
	{
		group.GET("", m.guard.Require(auth.PermRepairView, "repair.view", "repair"), m.handler.List)
		group.POST("", m.guard.Require(auth.PermRepairClaim, "repair.create", "repair"), m.handler.Create)
		group.GET("/meta", m.guard.Require(auth.PermRepairView, "repair.view", "repair"), m.handler.Metadata)
		group.GET("/statistics", m.guard.Require(auth.PermRepairView, "repair.view", "repair"), m.handler.Statistics)
		group.GET("/fault/:faultId", m.guard.Require(auth.PermRepairView, "repair.view", "repair"), m.handler.ListByFault)
		group.GET("/:id", m.guard.Require(auth.PermRepairView, "repair.view", "repair"), m.handler.Get)
		// 推进(编辑)维修记录: 有 repair:advance 即可, 但是否为本人负责的记录在处理器内二次判定。
		group.PUT("/:id", m.guard.Require(auth.PermRepairAdvance, "repair.update", "repair"), m.handler.Update)
		group.POST("/:id/finish", m.guard.Require(auth.PermRepairAdvance, "repair.finish", "repair"), m.handler.Finish)
		group.POST("/:id/reassign", m.guard.Require(auth.PermRepairReassign, "repair.reassign", "repair"), m.handler.Reassign)
		// 删除属于管理动作, 维修人员/登记人员均无权。
		group.DELETE("/:id", m.guard.Require(auth.PermRepairManage, "repair.delete", "repair"), m.handler.Delete)
	}
}
