import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/api/auth'

const TOKEN_KEY = 'streetlight_token'

// 认证状态: 令牌持久化在 localStorage, 当前用户与权限点来自后端 /auth/me。
// 页面、按钮、菜单的可见性一律用 can(权限点) 判断, 与后端接口/导出/看板同源,
// 前端不自行根据角色推导权限, 避免前后端结论不一致。
export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const profile = ref(null)
  const permissions = ref(new Set())
  const loaded = ref(false)

  const isLoggedIn = computed(() => !!token.value)
  const displayName = computed(() => profile.value?.display_name || '')
  const roleLabel = computed(() => profile.value?.role_label || '')
  const role = computed(() => profile.value?.role || '')

  function setToken(value) {
    token.value = value || ''
    if (value) {
      localStorage.setItem(TOKEN_KEY, value)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  function applyProfile(data) {
    profile.value = data
    permissions.value = new Set(data?.permissions || [])
    loaded.value = true
  }

  // can 判断当前用户是否拥有某权限点; 未加载完成时一律视为无权限(保守收敛)。
  function can(permission) {
    return permissions.value.has(permission)
  }

  // canAny 拥有任一权限即可。
  function canAny(list) {
    return (list || []).some((item) => permissions.value.has(item))
  }

  // isRepairman 用于行级按钮: 维修人员只能在"本人负责"的记录上看到推进按钮。
  const isRepairman = computed(() => role.value === 'repairman')
  const isManager = computed(() => role.value === 'manager')

  async function login(username, password) {
    const result = await authApi.login(username, password)
    setToken(result.token)
    applyProfile(result.user)
    return result.user
  }

  async function fetchMe() {
    if (!token.value) return null
    const data = await authApi.me()
    applyProfile(data.profile)
    return data.profile
  }

  async function logout() {
    try {
      await authApi.logout()
    } catch (error) {
      // 即便令牌已失效也要清理本地状态
    }
    reset()
  }

  function reset() {
    setToken('')
    profile.value = null
    permissions.value = new Set()
    loaded.value = false
  }

  // ownsRepair 判断某条维修记录是否为当前维修人员本人负责。
  // 管理岗与其它角色返回 true(其可见性由权限点控制), 维修人员必须负责人匹配。
  function ownsRepair(row) {
    if (role.value !== 'repairman') return true
    return !!row && row.repairman === profile.value?.display_name
  }

  return {
    token,
    profile,
    loaded,
    isLoggedIn,
    displayName,
    roleLabel,
    role,
    isRepairman,
    isManager,
    setToken,
    applyProfile,
    can,
    canAny,
    login,
    fetchMe,
    logout,
    reset,
    ownsRepair,
  }
})
