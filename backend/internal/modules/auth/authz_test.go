package auth

import (
	"net/http"
	"strings"
	"testing"

	"streetlight/internal/apperr"
)

func registrar() *Principal {
	return &Principal{ID: 10, Username: "reg01", DisplayName: "登记员小王", Role: RoleRegistrar}
}

func repairman() *Principal {
	return &Principal{ID: 20, Username: "rep01", DisplayName: "刘志强", Role: RoleRepairman, Team: "市政照明一班"}
}

func otherRepairman() *Principal {
	return &Principal{ID: 21, Username: "rep02", DisplayName: "陈鹏", Role: RoleRepairman, Team: "市政照明二班"}
}

func manager() *Principal {
	return &Principal{ID: 30, Username: "mgr01", DisplayName: "调度管理", Role: RoleManager}
}

// TestRolePermissionMatrix 固化三类角色的授权边界。
func TestRolePermissionMatrix(t *testing.T) {
	reg, rep, mgr := registrar(), repairman(), manager()

	// 登记人员: 只做登记和查询。
	mustCan := func(p *Principal, permission string, want bool) {
		t.Helper()
		if got := p.Can(permission); got != want {
			t.Fatalf("%s 对 %s 的授权期望 %v, 实际 %v", p.Role, permission, want, got)
		}
	}
	mustCan(reg, PermFaultCreate, true)
	mustCan(reg, PermFaultRead, true)
	mustCan(reg, PermRepairRead, true)
	mustCan(reg, PermRepairCreate, false)
	mustCan(reg, PermRepairFinish, false)
	mustCan(reg, PermLampManage, false)
	mustCan(reg, PermFaultClose, false)
	mustCan(reg, PermRepairAssign, false)
	mustCan(reg, PermFaultBypass, false)
	mustCan(reg, PermUserManage, false)

	// 维修人员: 只能推进维修记录, 不能登记故障、不能维护台账、不能改派/例外。
	mustCan(rep, PermRepairRead, true)
	mustCan(rep, PermRepairCreate, true)
	mustCan(rep, PermRepairFinish, true)
	mustCan(rep, PermFaultCreate, false)
	mustCan(rep, PermFaultClose, false)
	mustCan(rep, PermRepairAssign, false)
	mustCan(rep, PermFaultBypass, false)
	mustCan(rep, PermLampManage, false)
	mustCan(rep, PermExportRepair, true)
	mustCan(rep, PermExportFault, false)

	// 管理岗: 全部权限。
	for _, permission := range []string{
		PermLampManage, PermFaultClose, PermFaultDelete,
		PermRepairAssign, PermFaultBypass, PermUserManage, PermAuditRead,
	} {
		mustCan(mgr, permission, true)
	}
}

// TestRepairOwnScope 维修人员只能接触本人负责的记录, 其它角色为全部范围。
func TestRepairOwnScope(t *testing.T) {
	scope, name := repairman().RepairDataScope()
	if scope != ScopeOwn || name != "刘志强" {
		t.Fatalf("维修人员范围应为 own/刘志强, 实际 %s/%s", scope, name)
	}

	for _, p := range []*Principal{registrar(), manager(), nil} {
		scope, _ := p.RepairDataScope()
		if scope != ScopeAll {
			t.Fatalf("角色 %v 应为全部范围, 实际 %s", p, scope)
		}
	}

	if !repairman().OwnsRepair("刘志强") {
		t.Fatal("维修人员应拥有本人负责的记录")
	}
	if repairman().OwnsRepair("陈鹏") {
		t.Fatal("维修人员不应拥有他人负责的记录")
	}
	if manager().OwnsRepair("刘志强") {
		t.Fatal("管理岗不按负责人归属判定")
	}
	if otherRepairman().OwnsRepair("刘志强") {
		t.Fatal("同事之间不能互相操作对方记录")
	}
}

// TestDeniedErrorNamesMissingPermission 被拒操作必须明确指出缺少哪一项授权。
func TestDeniedErrorNamesMissingPermission(t *testing.T) {
	err := DeniedError{Permission: PermRepairFinish, Reason: "仅限本人负责的维修记录, 该记录负责人为 陈鹏"}
	message := err.Error()
	if !strings.Contains(message, PermRepairFinish) {
		t.Fatalf("拒绝信息应包含缺失权限点: %s", message)
	}
	if !strings.Contains(message, "完工本人负责的维修记录") {
		t.Fatalf("拒绝信息应包含权限中文含义: %s", message)
	}
	if !strings.Contains(message, "陈鹏") {
		t.Fatalf("拒绝信息应说明数据范围原因: %s", message)
	}

	// apperr.Forbidden 应映射为 403 并携带缺失权限项。
	business := apperr.Forbidden(PermFaultBypass, DeniedError{Permission: PermFaultBypass}.Error())
	if business.Status != http.StatusForbidden || business.MissingPermission != PermFaultBypass {
		t.Fatalf("403 错误应携带缺失权限项, 实际 status=%d perm=%s", business.Status, business.MissingPermission)
	}
}
