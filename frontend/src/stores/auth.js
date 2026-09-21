import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { authApi } from '@/api/auth'
import { tokenStore } from '@/api/request'

// useAuthStore 管理登录态、当前用户与权限集合。
// can() 是前端唯一的权限判定函数, 与后端 HasPermission 使用同一份权限点。
export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const permissions = ref([])
  const roles = ref([])
  const initialized = ref(false)

  const permissionSet = computed(() => new Set(permissions.value))

  const isLoggedIn = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  // can 判断是否拥有某个权限点(页面/按钮/路由共用)。
  function can(permission) {
    return permissionSet.value.has('*') || permissionSet.value.has(permission)
  }

  // canAny 拥有任意一个即可。
  function canAny(list) {
    if (permissionSet.value.has('*')) return true
    return list.some((item) => permissionSet.value.has(item))
  }

  function applySession(data) {
    if (data?.user) user.value = data.user
    if (Array.isArray(data?.permissions)) permissions.value = data.permissions
    if (Array.isArray(data?.roles)) roles.value = data.roles
  }

  async function login(username, password) {
    const data = await authApi.login({ username, password })
    tokenStore.set(data.token)
    applySession(data)
    return data
  }

  async function fetchProfile() {
    const data = await authApi.profile()
    applySession(data)
    return data


  }

  function logout() {
    return authApi
      .logout()
      .catch(() => {})
      .finally(() => {
        clearSession()
      })
  }

  function clearSession() {
    tokenStore.clear()
    user.value = null
    permissions.value = []
    roles.value = []
    initialized.value = false
  }

  // bootstrap 在应用启动时恢复会话(若本地有 token)。
  async function bootstrap() {
    if (initialized.value) return
    if (!tokenStore.get()) {
      initialized.value = true
      return
    }
    try {
      await fetchProfile()
    } catch (error) {
      clearSession()
    } finally {
      initialized.value = true
    }
  }

  return {
    user,
    permissions,
    roles,
    isLoggedIn,
    isAdmin,
    can,
    canAny,
    applySession,
    login,
    fetchProfile,
    logout,
    clearSession,
    bootstrap,
  }
})
