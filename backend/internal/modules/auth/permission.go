package auth

// 角色取值。
const (
	// RoleRegistrar 登记人员: 只做登记和查询。
	RoleRegistrar = "registrar"
	// RoleRepair 维修人员: 只推进自己负责的维修记录, 并可查询。
	RoleRepair = "repair"
	// RoleAdmin 管理岗: 维护台账、改派、例外流转、导出、看板与审计查看等全部操作。
	RoleAdmin = "admin"
)

// 权限点(授权)取值。命名约定 <资源>:<动作>, 是全系统唯一的权限标识:
// 页面、接口、导出、看板四处只能引用这里定义的常量, 不得自行拼写。
const (
	// 路灯台账
	PermLampView   = "lamp:view"
	PermLampManage = "lamp:manage" // 新增 / 修改 / 删除路灯

	// 故障登记
	PermFaultView      = "fault:view"
	PermFaultRegister  = "fault:register" // 登记故障
	PermFaultUpdate    = "fault:update"   // 修改故障登记信息
	PermFaultClose     = "fault:close"    // 关闭(闭环/作废)
	PermFaultDelete    = "fault:delete"
	PermFaultReassign  = "fault:reassign"  // 改派负责人
	PermFaultException = "fault:exception" // 例外流转(跳过常规状态机)

	// 维修记录
	PermRepairView      = "repair:view"
	PermRepairClaim     = "repair:claim"     // 维修人员认领/被派单后开工
	PermRepairAdvance   = "repair:advance"   // 推进(编辑/完工)自己负责的记录
	PermRepairManage    = "repair:manage"    // 对任意维修记录操作(改派、删除、非本人记录)
	PermRepairAssign    = "repair:assign"    // 录入维修记录时指定负责人(派工)
	PermRepairReassign  = "repair:reassign"  // 改派维修负责人
	PermRepairException = "repair:exception" // 例外流转(回退/强制状态跳转)

	// 查询看板与导出
	PermStatusView   = "status:view"   // 看板与状态查询
	PermExportFault  = "export:fault"  // 导出故障数据
	PermExportRepair = "export:repair" // 导出维修数据

	// 系统管理(管理岗)
	PermAuditView  = "audit:view" // 查看操作审计
	PermUserManage = "user:manage"
)

// PermissionMeta 描述一个权限点的展示信息, 供前端与审计页使用。
type PermissionMeta struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Resource string `json:"resource"`
}

// permissionCatalog 是权限点的展示目录(顺序即展示顺序)。
var permissionCatalog = []PermissionMeta{
	{PermLampView, "查看路灯台账", "路灯台账"},
	{PermLampManage, "维护路灯台账", "路灯台账"},
	{PermFaultView, "查看故障", "故障登记"},
	{PermFaultRegister, "登记故障", "故障登记"},
	{PermFaultUpdate, "修改故障", "故障登记"},
	{PermFaultClose, "关闭故障", "故障登记"},
	{PermFaultDelete, "删除故障", "故障登记"},
	{PermFaultReassign, "改派故障负责人", "故障登记"},
	{PermFaultException, "故障例外流转", "故障登记"},
	{PermRepairView, "查看维修记录", "维修记录"},
	{PermRepairClaim, "认领/开工", "维修记录"},
	{PermRepairAdvance, "推进本人负责的记录", "维修记录"},
	{PermRepairAssign, "派工(指定负责人)", "维修记录"},
	{PermRepairManage, "管理全部维修记录", "维修记录"},
	{PermRepairReassign, "改派维修负责人", "维修记录"},
	{PermRepairException, "维修例外流转", "维修记录"},
	{PermStatusView, "查看看板", "状态查询"},
	{PermExportFault, "导出故障数据", "数据导出"},
	{PermExportRepair, "导出维修数据", "数据导出"},
	{PermAuditView, "查看操作审计", "系统管理"},
	{PermUserManage, "用户与权限管理", "系统管理"},
}

// WildcardPermission 是管理岗持有的通配授权, 判定时等价于拥有全部权限点。
const WildcardPermission = "*"

// rolePermissions 是 "角色 -> 权限点" 的唯一映射表, 是全系统权限判定的单一事实源。
var rolePermissions = map[string][]string{
	// 登记人员: 只做登记和查询, 不能改/关/删, 不能碰维修推进、改派、例外、导出。
	RoleRegistrar: {
		PermLampView,
		PermFaultView,
		PermFaultRegister,
		PermRepairView,
		PermStatusView,
	},
	// 维修人员: 只推进自己负责的维修记录(资源级归属在处理器内二次判定), 其余只读。
	RoleRepair: {
		PermLampView,
		PermFaultView,
		PermRepairView,
		PermRepairClaim,
		PermRepairAdvance,
		PermStatusView,
	},
	// 管理岗: 台账维护、故障修改/关闭/删除、改派、例外流转、导出、审计、用户管理等全部操作。
	RoleAdmin: {
		WildcardPermission,
	},
}

// RoleMeta 描述角色展示信息。
type RoleMeta struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// roleCatalog 角色目录(顺序即展示顺序)。
var roleCatalog = []RoleMeta{
	{RoleRegistrar, "登记人员", "只做故障登记与信息查询"},
	{RoleRepair, "维修人员", "只推进自己负责的维修记录"},
	{RoleAdmin, "管理岗", "台账维护、改派、例外流转、导出与审计"},
}

// PermissionsForRole 返回某角色拥有的权限点; admin 返回通配符。
// 返回的切片是副本, 调用方不得据此修改全局映射。
func PermissionsForRole(role string) []string {
	permissions, ok := rolePermissions[role]
	if !ok {
		return []string{}
	}
	out := make([]string, len(permissions))
	copy(out, permissions)
	return out
}

// HasPermission 判断角色是否被授予指定权限点。这是页面、接口、导出、看板
// 四处共用的同一个判定函数, 不允许在其它地方另写角色判断逻辑。
func HasPermission(role, permission string) bool {
	if role == "" || permission == "" {
		return false
	}
	for _, granted := range rolePermissions[role] {
		if granted == WildcardPermission || granted == permission {
			return true
		}
	}
	return false
}

// RoleCatalog 返回角色目录副本。
func RoleCatalog() []RoleMeta {
	out := make([]RoleMeta, len(roleCatalog))
	copy(out, roleCatalog)
	return out
}

// PermissionCatalog 返回权限点目录副本。
func PermissionCatalog() []PermissionMeta {
	out := make([]PermissionMeta, len(permissionCatalog))
	copy(out, permissionCatalog)
	return out
}

// RoleLabel 返回角色中文名, 未知角色回退为原值。
func RoleLabel(role string) string {
	for _, item := range roleCatalog {
		if item.Key == role {
			return item.Label
		}
	}
	return role
}

// IsValidRole 校验角色取值。
func IsValidRole(role string) bool {
	_, ok := rolePermissions[role]
	return ok
}

// PermissionLabel 返回权限点中文名, 未知权限返回空串。
func PermissionLabel(permission string) string {
	for _, item := range permissionCatalog {
		if item.Key == permission {
			return item.Label
		}
	}
	return ""
}
