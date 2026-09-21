package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRolePermissionsAreSeparated(t *testing.T) {
	// 登记人员: 只做登记和查询, 不能维修推进 / 改派 / 例外 / 导出 / 审计。
	registrar := RoleRegistrar
	require.True(t, HasPermission(registrar, PermFaultRegister))
	require.True(t, HasPermission(registrar, PermFaultView))
	require.False(t, HasPermission(registrar, PermLampManage), "登记人员不能维护台账")
	require.False(t, HasPermission(registrar, PermRepairAdvance), "登记人员不能推进维修")
	require.False(t, HasPermission(registrar, PermFaultReassign))
	require.False(t, HasPermission(registrar, PermFaultException))
	require.False(t, HasPermission(registrar, PermExportFault))
	require.False(t, HasPermission(registrar, PermAuditView))

	// 维修人员: 只能推进自己负责的记录, 不能登记故障 / 改派 / 例外 / 删除。
	repairman := RoleRepair
	require.True(t, HasPermission(repairman, PermRepairAdvance))
	require.True(t, HasPermission(repairman, PermRepairClaim))
	require.False(t, HasPermission(repairman, PermFaultRegister), "维修人员不能登记故障")
	require.False(t, HasPermission(repairman, PermRepairManage), "维修人员不能删除/管理任意记录")
	require.False(t, HasPermission(repairman, PermRepairReassign))
	require.False(t, HasPermission(repairman, PermRepairException))
	require.False(t, HasPermission(repairman, PermExportRepair))

	// 管理岗: 通配授权拥有全部权限点。
	admin := RoleAdmin
	for _, perm := range PermissionCatalog() {
		require.True(t, HasPermission(admin, perm.Key), "管理岗应拥有 %s", perm.Key)
	}
	require.True(t, HasPermission(admin, PermFaultReassign))
	require.True(t, HasPermission(admin, PermFaultException))
}

func TestUnknownRoleAndEmptyInput(t *testing.T) {
	require.False(t, HasPermission("ghost", PermFaultView))
	require.False(t, HasPermission(RoleAdmin, ""))
	require.False(t, HasPermission("", PermFaultView))
}

func TestPermissionCatalogCoversAllGuardedPoints(t *testing.T) {
	// 保证目录里能查到每个被守卫引用的权限点的中文名。
	for _, key := range []string{
		PermLampView, PermLampManage,
		PermFaultView, PermFaultRegister, PermFaultUpdate, PermFaultClose,
		PermFaultDelete, PermFaultReassign, PermFaultException,
		PermRepairView, PermRepairClaim, PermRepairAdvance, PermRepairManage,
		PermRepairAssign, PermRepairReassign, PermRepairException,
		PermStatusView, PermExportFault, PermExportRepair,
		PermAuditView, PermUserManage,
	} {
		require.NotEmpty(t, PermissionLabel(key), "权限点 %s 缺少中文目录", key)
	}
}
