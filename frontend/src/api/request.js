import axios from 'axios'
import { ElMessage } from 'element-plus'

// 统一的 axios 实例: 后端始终返回 { code, message, data } 结构。
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
})

// 成功响应直接返回 data, 让调用方只关心业务数据。
request.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 'OK') {
        return body.data
      }
      const error = new Error(body.message || '请求失败')
      error.code = body.code
      if (!response.config?.silent) {
        ElMessage.error(error.message)
      }
      return Promise.reject(error)
    }
    return body
  },
  (error) => {
    const payload = error.response?.data
    const normalized = new Error(payload?.message || error.message || '网络异常, 请稍后重试')
    normalized.code = payload?.code || 'NETWORK_ERROR'
    normalized.status = error.response?.status
    if (!error.config?.silent) {
      ElMessage.error(normalized.message)
    }
    return Promise.reject(normalized)
  },
)

export default request
