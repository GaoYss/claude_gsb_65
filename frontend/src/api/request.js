import axios from 'axios'
import { ElMessage } from 'element-plus'

const TOKEN_KEY = 'streetlight_token'

// 统一的 axios 实例: 后端始终返回 { code, message, data, missing_permission } 结构。
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
})

// 请求注入登录令牌。
request.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 跳转登录页(避免直接依赖 router 形成循环引用)。
function redirectToLogin() {
  const current = window.location.pathname + window.location.search
  if (!window.location.pathname.startsWith('/login')) {
    window.location.assign(`/login?redirect=${encodeURIComponent(current)}`)
  }
}

// 成功响应直接返回 data, 让调用方只关心业务数据。
request.interceptors.response.use(
  (response) => {
    // 文件流(导出)直接原样返回, 不走信封解包。
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
      error.missingPermission = body.missing_permission
      if (!response.config?.silent) {
        ElMessage.error(error.message)
      }
      return Promise.reject(error)
    }
    return body
  },
  async (error) => {
    const status = error.response?.status
    let payload = error.response?.data

    // 导出(blob)出错时, 错误体同样是 blob, 需要先读出 JSON 才能拿到统一信封。
    if (payload instanceof Blob && payload.type && payload.type.includes('application/json')) {
      try {
        payload = JSON.parse(await payload.text())
      } catch (error) {
        payload = undefined
      }
    }

    const normalized = new Error(payload?.message || error.message || '网络异常, 请稍后重试')
    normalized.code = payload?.code || 'NETWORK_ERROR'
    normalized.status = status
    normalized.missingPermission = payload?.missing_permission

    if (status === 401) {
      // 登录接口本身的 401 由登录页自行提示, 不触发跳转。
      if (!error.config?.silentAuth) {
        localStorage.removeItem(TOKEN_KEY)
        if (!error.config?.silent) {
          ElMessage.error(normalized.message || '登录状态已失效, 请重新登录')
        }
        redirectToLogin()
      }
    } else if (!error.config?.silent) {
      // 403 时后端已在 message 中写明缺少的授权项, 直接展示同一条结论。
      ElMessage.error(normalized.message)
    }
    return Promise.reject(normalized)
  },
)

export default request
