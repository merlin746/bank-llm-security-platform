import request from './request'
import { useMock, mockLogin, mockCurrentUser, mockLogout } from '@/mock'

/**
 * 用户登录（组员 B 业务后台）
 * POST /api/auth/login
 * 请求体: { username, password }
 * 响应:   { code, data: { token, user: { username, role, dataLevel } }, msg }
 */
export function login(payload) {
  if (useMock()) return mockLogin(payload)
  return request.post('/auth/login', payload)
}

/**
 * 退出登录
 * POST /api/auth/logout
 */
export function logout(token) {
  if (useMock()) return mockLogout(token)
  return request.post('/auth/logout', null, { headers: { Authorization: `Bearer ${token}` } })
}

export function currentUser() {
  if (useMock()) return mockCurrentUser()
  return request.get('/auth/me')
}
