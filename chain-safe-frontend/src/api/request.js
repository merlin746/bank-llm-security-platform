import axios from 'axios'

/**
 * 统一 axios 实例：
 * - baseURL 指向 /api，开发态由 vite proxy 转发到 Go 后端网关（127.0.0.1:8080）
 * - 自动附带 Authorization: Bearer <token>
 * - 统一响应拦截：后端约定返回 { code, data, msg }，code === 0 为成功
 * - 401 时跳转登录页
 */
const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 10000
})

request.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

request.interceptors.response.use(
  (res) => {
    const body = res.data
    // 后端统一格式 { code, data, msg }
    if (body && typeof body.code !== 'undefined' && body.code !== 0) {
      return Promise.reject(new Error(body.msg || '业务处理失败'))
    }
    return body
  },
  (err) => {
    const status = err.response?.status
    const msg = err.response?.data?.msg || err.message || '网络请求失败'
    if (status === 401) {
      window.location.href = '/login'
    }
    return Promise.reject(new Error(msg))
  }
)

export default request
