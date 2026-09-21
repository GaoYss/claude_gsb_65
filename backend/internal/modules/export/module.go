package export

import (
	"github.com/gin-gonic/gin"

	"streetlight/internal/module"
	"streetlight/internal/modules/auth"
)

// Module 数据导出模块, 自身不建表, 复用 fault/repair 的列表服务与同一套权限/数据范围。
type Module struct {
	service *Service
	handler *Handler
	guard   *auth.Guard
}

// New 构造导出模块。
func New(faults FaultLister, repairs RepairLister, guard *auth.Guard) *Module {
	service := NewService(faults, repairs)
	return &Module{service: service, handler: NewHandler(service, guard), guard: guard}
}

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "数据导出" }

// Models 实现 module.Module 接口: 导出模块无数据表。
func (m *Module) Models() []any { return nil }

// RegisterRoutes 实现 module.Module 接口。
// 导出与页面、接口、看板使用同一个 Guard 判定, 越权调用会在中间件被拒绝并留痕。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/export")
	{
		group.GET("/faults",
			m.guard.Authenticate(true),
			m.guard.Require(auth.PermExportFault, "export.fault", "export"),
			m.handler.ExportFaults)
		group.GET("/repairs",
			m.guard.Authenticate(true),
			m.guard.Require(auth.PermExportRepair, "export.repair", "export"),
			m.handler.ExportRepairs)
	}
}

var _ module.Module = (*Module)(nil)
