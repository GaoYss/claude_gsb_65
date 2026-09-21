package auth

import "fmt"

// 权限点采用 "资源:动作" 命名。页面按钮显隐、接口放行、导出范围、看板数据范围
// 全部引用同一组常量, 保证四处结论一致。
const (
	PermLampRead   = "lamp:read"   // 查看路灯台账
	PermLampManage = "lamp:manage" // 维护路灯台账(增改删)

	PermFaultRead   = "fault:read"   // 查询故障
	PermFaultCreate = "fault:create" // 登记故障
	PermFaultUpdate = "fault:update" // 修改故障登记信息
	PermFaultClose  = "fault:close"  // 关闭/作废故障
	PermFaultDelete = "fault:delete" // 删除故障
	PermFaultBypass = "fault:bypass" // 例外流转: 跳过常规状态机约束强制推进或回退故障

	PermRepairRead   = "repair:read"   // 查询维修记录
	PermRepairCreate = "repair:create" // 开工登记维修记录
	PermRepairUpdate = "repair:update" // 推进/编辑本人负责的维修记录
	PermRepairFinish = "repair:finish" // 完工本人负责的维修记录
	PermRepairDelete = "repair:delete" // 删除维修记录
	PermRepairAssign = "repair:assign" // 改派: 调整维修记录的负责人或班组

	PermStatusRead = "status:read" // 查看运行看板与状态查询

	PermExportFault  = "export:fault"  // 导出故障数据
	PermExportRepair = "export:repair" // 导出维修数据(本人范围)

	PermUserManage = "user:manage" // 用户与角色授权管理
	PermAuditRead  = "audit:read"  // 查看审计日志
)

// PermissionInfo 描述权限点的中文含义, 用于越权提示与前端渲染。
var PermissionInfo = map[string]string{
	PermLampRead:     "查看路灯台账",
	PermLampManage:   "维护路灯台账",
	PermFaultRead:    "查询故障",
	PermFaultCreate:  "登记故障",
	PermFaultUpdate:  "修改故障登记信息",
	PermFaultClose:   "关闭/作废故障",
	PermFaultDelete:  "删除故障",
	PermFaultBypass:  "例外流转故障状态",
	PermRepairRead:   "查询维修记录",
	PermRepairCreate: "开工登记维修记录",
	PermRepairUpdate: "推进本人负责的维修记录",
	PermRepairFinish: "完工本人负责的维修记录",
	PermRepairDelete: "删除维修记录",
	PermRepairAssign: "改派维修负责人",
	PermStatusRead:   "查看运行看板",
	PermExportFault:  "导出故障数据",
	PermExportRepair: "导出维修数据",
	PermUserManage:   "用户与授权管理",
	PermAuditRead:    "查看审计日志",
}

// rolePermissions 是角色到权限集合的唯一映射, 任何地方都不允许另行为角色开口子。
var rolePermissions = map[string]map[string]struct{}{
	RoleRegistrar: toSet(
		PermLampRead,
		PermFaultRead, PermFaultCreate, PermFaultUpdate,
		PermRepairRead,
		PermStatusRead,
		PermExportFault,
	),
	RoleRepairman: toSet(
		PermLampRead,
		PermFaultRead,
		PermRepairRead, PermRepairCreate, PermRepairUpdate, PermRepairFinish,
		PermStatusRead,
		PermExportRepair,
	),
	RoleManager: toSet(
		PermLampRead, PermLampManage,
		PermFaultRead, PermFaultCreate, PermFaultUpdate, PermFaultClose, PermFaultDelete,
		PermFaultBypass,
		PermRepairRead, PermRepairCreate, PermRepairUpdate, PermRepairFinish, PermRepairDelete,
		PermRepairAssign,
		PermStatusRead,
		PermExportFault, PermExportRepair,
		PermUserManage, PermAuditRead,
	),
}

// ScopeAll 表示数据范围为全部; ScopeOwn 表示仅限本人负责的数据。
const (
	ScopeAll = "all"
	ScopeOwn = "own"
)

func toSet(items ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		set[item] = struct{}{}
	}
	return set
}

// PermissionsOf 返回某角色拥有的全部权限点, 结果为新切片, 调用方修改不影响策略表。
func PermissionsOf(role string) []string {
	set := rolePermissions[role]
	result := make([]string, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	return result
}

// Principal 是请求上下文中的当前操作者, 由认证中间件写入。
type Principal struct {
	ID          uint
	Username    string
	DisplayName string
	Role        string
	Team        string
}

// Can 判断是否拥有指定权限点。
func (p *Principal) Can(permission string) bool {
	if p == nil {
		return false
	}
	set, ok := rolePermissions[p.Role]
	if !ok {
		return false
	}
	_, ok = set[permission]
	return ok
}

// Permissions 返回当前操作者的权限点集合(供 /auth/me 与前端按钮显隐使用)。
func (p *Principal) Permissions() []string {
	return PermissionsOf(p.Role)
}

// RepairmanIdentity 返回维修人员在维修记录中的归属标识(即维修人姓名)。
func (p *Principal) RepairmanIdentity() string { return p.DisplayName }

// RepairDataScope 返回维修数据的可见范围:
// 维修人员仅能看到本人负责的记录; 登记人员/管理岗可见全部(登记人员只读)。
func (p *Principal) RepairDataScope() (scope string, repairman string) {
	if p != nil && p.Role == RoleRepairman {
		return ScopeOwn, p.DisplayName
	}
	return ScopeAll, ""
}

// OwnsRepair 判断当前操作者是否为该维修记录的负责人。
// 管理岗与非维修角色不按负责人归属(他们的可见性由权限点另行控制)。
func (p *Principal) OwnsRepair(repairman string) bool {
	if p == nil || p.Role != RoleRepairman {
		return false
	}
	return p.DisplayName != "" && p.DisplayName == repairman
}

// DeniedError 描述一次越权拒绝, 必须明确指出缺少的授权项。
type DeniedError struct {
	Permission string // 缺失的权限点
	Reason     string // 补充原因, 例如 "仅限本人负责的记录"
}

// Error 实现 error, 文案直接面向使用者, 指明缺少哪一项授权。
func (e DeniedError) Error() string {
	label := PermissionInfo[e.Permission]
	if label == "" {
		label = e.Permission
	}
	if e.Reason != "" {
		return fmt.Sprintf("无权执行该操作: 缺少授权 %s(%s), %s", e.Permission, label, e.Reason)
	}
	return fmt.Sprintf("无权执行该操作: 缺少授权 %s(%s)", e.Permission, label)
}
