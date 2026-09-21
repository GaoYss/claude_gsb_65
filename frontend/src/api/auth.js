import request from './request'

// 认证、用户管理与审计日志接口。
export const authApi = {
  login: (username, password) =>
    request.post('/auth/login', { username, password }, { silent: true, silentAuth: true }),
  logout: () => request.post('/auth/logout'),
  me: () => request.get('/auth/me'),

  // 用户管理(管理岗)
  listUsers: (params) => request.get('/auth/users', { params }),
  createUser: (data) => request.post('/auth/users', data),
  updateUser: (id, data) => request.put(`/auth/users/${id}`, data),

  // 审计日志(管理岗)
  listAuditLogs: (params) => request.get('/auth/audit-logs', { params }),
  permissionMeta: () => request.get('/auth/permissions'),
}
