package repair_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/auth"
	"streetlight/internal/modules/repair"
)

func repairmanCtx(name string) context.Context {
	return auth.WithPrincipal(context.Background(), &auth.Principal{
		ID: 20, Username: "rep_" + name, DisplayName: name, Role: auth.RoleRepairman,
	})
}

// TestRepairmanOwnershipEnforced 验证维修人员只能推进本人负责的记录,
// 且列表/详情/完工都遵守同一数据范围, 拒绝时指明缺失权限项。
func TestRepairmanOwnershipEnforced(t *testing.T) {
	mgr := managerCtx()
	h := newHarness(t)
	device := h.createLamp(t, "LD-OWN-001")
	faultEntity := h.createFault(t, device.ID, "归属校验用故障")

	// 管理岗代 刘志强 开工。
	record, err := h.repairs.Create(mgr, repair.CreateRequest{
		FaultID: faultEntity.ID, Repairman: "刘志强", RepairTeam: "市政照明一班",
	})
	require.NoError(t, err)

	liuzq := repairmanCtx("刘志强")
	chenpeng := repairmanCtx("陈鹏")

	// 本人列表应只看到自己的记录。
	items, total, _, err := h.repairs.List(liuzq, repair.ListQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, "刘志强", items[0].Repairman)

	// 同事列表看不到该记录。
	_, otherTotal, _, err := h.repairs.List(chenpeng, repair.ListQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(0), otherTotal)

	// 同事直接按 ID 查详情 -> 403, 而不是 404(已通过归属校验阶段)。
	_, err = h.repairs.Get(chenpeng, record.ID)
	requireForbidden(t, err, auth.PermRepairRead)

	// 同事尝试完工他人记录 -> 403 且指明缺 repair:finish。
	_, err = h.repairs.Finish(chenpeng, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	requireForbidden(t, err, auth.PermRepairFinish)

	// 登记人员连完工权限点都没有。
	registrar := auth.WithPrincipal(context.Background(), &auth.Principal{
		ID: 40, Username: "reg01", DisplayName: "登记员", Role: auth.RoleRegistrar,
	})
	_, err = h.repairs.Finish(registrar, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	requireForbidden(t, err, auth.PermRepairFinish)

	// 本人可以完工自己的记录。
	finished, err := h.repairs.Finish(liuzq, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)
	require.Equal(t, repair.StatusFinished, finished.Status)
}

// TestRepairmanCannotCreateForOthers 维修人员不能把开工记录记在别人名下。
func TestRepairmanCannotCreateForOthers(t *testing.T) {
	h := newHarness(t)
	device := h.createLamp(t, "LD-OWN-002")
	faultEntity := h.createFault(t, device.ID, "代登记拦截用故障")

	_, err := h.repairs.Create(repairmanCtx("刘志强"), repair.CreateRequest{
		FaultID: faultEntity.ID, Repairman: "陈鹏",
	})
	requireForbidden(t, err, auth.PermRepairCreate)
}

func requireForbidden(t *testing.T, err error, permission string) {
	t.Helper()
	require.Error(t, err)
	businessErr, ok := apperr.As(err)
	require.True(t, ok, "期望业务错误, 实际: %v", err)
	require.Equal(t, http.StatusForbidden, businessErr.Status, "错误信息: %s", businessErr.Message)
	require.Equal(t, permission, businessErr.MissingPermission, "拒绝应指明缺失的授权项")
}
