import { useAuthStore } from '@/stores/auth'

// v-permission 依据当前用户权限点控制元素是否渲染。
//
// 用法:
//   v-permission="'fault:register'"                              拥有该权限才渲染
//   v-permission="['repair:manage', 'repair:reassign']"          拥有任意一个即渲染
//
// 仅用于页面可用性(收敛入口); 真正的安全边界在后端同一判定, 即使绕过前端
// 直接调接口, 越权仍会被拒绝并写入审计日志。
export const permissionDirective = {
  mounted(el, binding) {
    if (!evaluate(binding.value)) {
      el.parentNode?.removeChild(el)
    }
  },
}

function evaluate(value) {
  const auth = useAuthStore()
  if (!value) return true
  if (Array.isArray(value)) {
    return value.some((item) => auth.can(item))
  }
  if (typeof value === 'string') {
    return auth.can(value)
  }
  return true
}
