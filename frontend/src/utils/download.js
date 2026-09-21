import { ElMessage } from 'element-plus'
import request from '@/api/request'

// downloadCsv 携带登录令牌请求导出接口并触发浏览器下载。
// 数据范围(如维修人员仅本人记录)由后端与列表/看板同一函数裁决, 前端不做范围拼接。
// 403 等错误后端返回的是 JSON 而非文件, 这里识别后展示缺失权限提示。
export async function downloadCsv(url, params, fallbackName = 'export.csv') {
  let response
  try {
    response = await request.get(url, { params, responseType: 'blob' })
  } catch (error) {
    // 拦截器已对 401/403 给出统一提示。
    return false
  }

  const blob = response.data
  // 错误场景: 后端返回 JSON(blob), 解析出统一错误信息。
  if (blob.type && blob.type.includes('application/json')) {
    try {
      const text = await blob.text()
      const data = JSON.parse(text)
      ElMessage.error(data.message || '导出失败')
    } catch (error) {
      ElMessage.error('导出失败')
    }
    return false
  }

  const disposition = response.headers?.['content-disposition'] || ''
  const matched = disposition.match(/filename\*?=(?:UTF-8'')?["']?([^;"']+)/i)
  const filename = matched ? decodeURIComponent(matched[1]) : fallbackName

  const link = document.createElement('a')
  const objectUrl = window.URL.createObjectURL(blob)
  link.href = objectUrl
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(objectUrl)
  return true
}
