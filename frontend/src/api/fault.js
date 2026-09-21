import request from './request'

// 故障登记接口。
export const faultApi = {
  list: (params) => request.get('/faults', { params }),
  detail: (id) => request.get(`/faults/${id}`),
  create: (data) => request.post('/faults', data),
  update: (id, data) => request.put(`/faults/${id}`, data),
  remove: (id) => request.delete(`/faults/${id}`),
  close: (id, data) => request.post(`/faults/${id}/close`, data),
  // 例外流转(仅管理岗)
  transition: (id, data) => request.post(`/faults/${id}/transition`, data),
  meta: () => request.get('/faults/meta'),
  // 导出 CSV(携带登录令牌, 以 blob 下载)
  exportUrl: '/faults/export',
}
