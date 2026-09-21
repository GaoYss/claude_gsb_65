import { useAuthStore } from '@/stores/auth'

// usePermission 暴露统一的权限判断, 页面/按钮/菜单都通过它决定可见性,
// 结论与后端接口、导出、看板完全同源(后端 /auth/me 返回的权限点)。
export function usePermission() {
  const auth = useAuthStore()
  return {
    can: (permission) => auth.can(permission),
    canAny: (list) => auth.canAny(list),
    ownsRepair: (row) => auth.ownsRepair(row),
  }
}
