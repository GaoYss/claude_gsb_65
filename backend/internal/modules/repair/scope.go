package repair

import (
	"context"

	"streetlight/internal/modules/auth"
)

// ScopeFromContext 返回当前操作者的维修数据范围, 是
// 页面列表 / 接口详情 / 导出 / 看板四处共用的唯一判定:
// 维修人员仅能接触本人负责(repairman 等于本人姓名)的记录;
// 登记人员与管理岗可见全部(登记人员只读, 写操作由各自权限点另行拦截)。
//
// 任何 handler/service 都不得另写一套 "repairman 角色" 判断, 一律调用本函数。
func ScopeFromContext(ctx context.Context) (scope string, ownRepairman string) {
	principal := auth.PrincipalFromContext(ctx)
	return principal.RepairDataScope()
}

// ApplyOwnScope 把当前操作者的数据范围叠加到查询过滤条件上:
// 维修人员强制只查本人记录, 其余角色不追加条件。
func ApplyOwnScope(ctx context.Context, filter Filter) Filter {
	scope, repairman := ScopeFromContext(ctx)
	if scope == auth.ScopeOwn {
		filter.Repairman = repairman
	}
	return filter
}

// principal 从上下文取当前操作者。
func principalFrom(ctx context.Context) *auth.Principal {
	return auth.PrincipalFromContext(ctx)
}

// ensureCanAdvance 校验当前操作者是否有权操作指定维修记录:
// 首先必须持有该权限点(登记人员等无此权限者直接被拒);
// 维修人员还必须是记录负责人(仅限本人负责的记录); 管理岗可操作任意记录。
// 返回的 error 已带缺失权限点, 直接上抛即可。
func ensureCanAdvance(ctx context.Context, entity *Repair, permission string) error {
	principal := principalFrom(ctx)
	if principal == nil {
		return auth.Forbidden(permission, "请先登录")
	}
	if !principal.Can(permission) {
		return auth.Forbidden(permission, "")
	}
	if principal.Role == auth.RoleRepairman && !principal.OwnsRepair(entity.Repairman) {
		return auth.Forbidden(permission, "仅限本人负责的维修记录, 该记录负责人为 "+entity.Repairman)
	}
	return nil
}

// ensureCanRead 校验当前操作者能否查看指定维修记录, 规则与推进一致, 保证
// "列表 / 详情 / 导出 / 看板" 对维修人员都是同一套本人范围结论。
func ensureCanRead(ctx context.Context, entity *Repair) error {
	return ensureCanAdvance(ctx, entity, auth.PermRepairRead)
}

// ensureCanCreateAs 校验开工登记时填写的负责人:
// 维修人员只能为本人开工(不能把活记在别人名下); 管理岗可代任意人登记。
func ensureCanCreateAs(ctx context.Context, repairman string) error {
	principal := principalFrom(ctx)
	if principal == nil {
		return auth.Forbidden(auth.PermRepairCreate, "请先登录")
	}
	if !principal.Can(auth.PermRepairCreate) {
		return auth.Forbidden(auth.PermRepairCreate, "")
	}
	if principal.Role == auth.RoleRepairman && principal.DisplayName != repairman {
		return auth.Forbidden(auth.PermRepairCreate, "维修人员仅能为本人开工登记, 不能指派给 "+repairman)
	}
	return nil
}
