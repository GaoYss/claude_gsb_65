import request from './request'

// 认证、用户管理与审计接口。
export const authApi = {
  login: (data) => request.post('/auth/login', data, { silent: true }),
  logout: () => request.post('/auth/logout'),
  profile: () => request.get('/auth/profile', { silent: true }),
  catalog: () => request.get('/auth/catalog', { silent: true }),
  changePassword: (data) => request.put('/auth/password', data),
}

export const userApi = {
  list: (params) => request.get('/users', { params }),
  create: (data) => request.post('/users', data),
  update: (id, data) => request.put(`/users/${id}`, data),
  resetPassword: (id, data) => request.post(`/users/${id}/reset-password`, data),
}

export const auditApi = {
  list: (params) => request.get('/audit-logs', { params }),
}
