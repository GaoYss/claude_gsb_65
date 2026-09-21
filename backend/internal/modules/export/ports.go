package export

import (
	"context"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/repair"
	"streetlight/pkg/pagination"
)

// FaultLister 故障列表端口, 由 fault.Service 实现。
// 直接复用业务模块的列表服务, 使导出与页面/接口走完全相同的过滤与数据范围判定。
type FaultLister interface {
	List(ctx context.Context, query fault.ListQuery) ([]fault.Fault, int64, pagination.Query, error)
}

// RepairLister 维修列表端口, 由 repair.Service 实现(维修人员仅能导出本人记录)。
type RepairLister interface {
	List(ctx context.Context, query repair.ListQuery) ([]repair.Repair, int64, pagination.Query, error)
}
