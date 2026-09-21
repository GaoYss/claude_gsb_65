import axios from 'axios'
import { ElMessage } from 'element-plus'

const TOKEN_KEY = 'streetlight_token'

export const tokenStore = {
  get: () => localStorage.getItem(TOKEN_KEY) || '',
  set: (token) => localStorage.setItem(TOKEN_KEY, token),
  clear: () => localStorage.removeItem(TOKEN_KEY),
}

// 统一的 axios 实例: 后端始终返回 { code, message, data, details? } 结构。
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
})

// 请求拦截: 自动附带登录令牌。
request.interceptors.request.use((config) => {
  const token = tokenStore.get()
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 成功响应直接返回 data, 让调用方只关心业务数据。
request.interceptors.response.use(
  (response) => {
    // 文件下载(blob)透传完整响应, 以便读取文件名等响应头。
    if (response.config?.responseType === 'blob') {
      return response
    }
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 'OK') {
        return body.data
      }
      const error = new Error(body.message || '请求失败')
      error.code = body.code
      error.details = body.details
      if (!response.config?.silent) {
        ElMessage.error(error.message)
      }
      return Promise.reject(error)
    }
    return body
  },
  async (error) => {
    let payload = error.response?.data

    // 文件下载请求被拒(如 403)时, 错误体也是 Blob, 需要读出其中的 JSON。
    if (payload instanceof Blob && payload.type && payload.type.includes('json')) {
      try {
        payload = JSON.parse(await payload.text())
      } catch {
        payload = undefined
      }
    }

    const status = error.response?.status
    const normalized = new Error(payload?.message || error.message || '网络异常, 请稍后重试')
    normalized.code = payload?.code || 'NETWORK_ERROR'
    normalized.details = payload?.details
    normalized.status = status

    // 401: 登录态失效, 清除令牌并跳转登录页(避免在登录页重复提示)。
    if (status === 401) {
      tokenStore.clear()
      if (!error.config?.silent && !window.location.hash.startsWith('#/login') && window.location.pathname !== '/login') {
        ElMessage.error(normalized.message || '登录已失效, 请重新登录')
      }
      if (!window.location.pathname.endsWith('/login')) {
        const redirect = encodeURIComponent(window.location.pathname + window.location.search)
        window.location.assign(`${window.location.pathname.replace(/\/[^/]*$/, '/login')}?redirect=${redirect}`)
      }
    } else if (!error.config?.silent) {
      ElMessage.error(normalized.message)
    }
    return Promise.reject(normalized)
  },
)

export default request
