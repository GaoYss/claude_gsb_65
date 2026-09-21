import request from './request'

// 数据导出。导出与列表页走同一套权限与数据范围(后端复用同一列表服务)。
async function download(path, params) {
  const response = await request.get(path, { params, responseType: 'blob' })
  const blob = new Blob([response.data], { type: 'text/csv;charset=utf-8' })
  const disposition = response.headers?.['content-disposition'] || ''
  const matched = disposition.match(/filename=([^;]+)/)
  const fileName = matched ? matched[1] : `${path.split('/').pop()}.csv`
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(url)
}

export const exportApi = {
  faults: (params) => download('/export/faults', params),
  repairs: (params) => download('/export/repairs', params),
}
