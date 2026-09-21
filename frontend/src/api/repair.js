import request from './request'

// 维修记录接口。
export const repairApi = {
  list: (params) => request.get('/repairs', { params }),
  detail: (id) => request.get(`/repairs/${id}`),
  listByFault: (faultId) => request.get(`/repairs/fault/${faultId}`),
  create: (data) => request.post('/repairs', data),
  update: (id, data) => request.put(`/repairs/${id}`, data),
  finish: (id, data) => request.post(`/repairs/${id}/finish`, data),
  // 改派(仅管理岗)
  assign: (id, data) => request.post(`/repairs/${id}/assign`, data),
  remove: (id) => request.delete(`/repairs/${id}`),
  meta: () => request.get('/repairs/meta'),
  statistics: () => request.get('/repairs/statistics'),
  // 导出 CSV(维修人员仅本人范围, 由后端裁决)
  exportUrl: '/repairs/export',
}
