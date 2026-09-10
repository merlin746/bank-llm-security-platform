import request from './request'
import { useMock, mockUsers, mockRoles, mockDataLevels } from '@/mock'

/**
 * 业务后台 CRUD（组员 B）：用户 / 角色 / 数据分级配置
 * 接口路径前缀统一为 /api
 */

// 用户列表 GET /api/users
export function getUsers() {
  if (useMock()) return mockUsers()
  return request.get('/users')
}

// 新增用户 POST /api/users
export function createUser(data) {
  return request.post('/users', data)
}

// 更新用户 PUT /api/users/:id
export function updateUser(id, data) {
  return request.put(`/users/${id}`, data)
}

// 删除用户 DELETE /api/users/:id
export function deleteUser(id) {
  return request.delete(`/users/${id}`)
}

// 角色列表 GET /api/roles
export function getRoles() {
  if (useMock()) return mockRoles()
  return request.get('/roles')
}

// 数据分级列表 GET /api/data-levels
export function getDataLevels() {
  if (useMock()) return mockDataLevels()
  return request.get('/data-levels')
}
