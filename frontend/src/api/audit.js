import request from './request'
import { useMock, mockTopology, mockAlerts, mockAlertDetail, mockRequestDetail, mockAuditRequests } from '@/mock'

/**
 * 审计溯源对账拓扑（组员 A 提供）
 * GET /api/audit/topology
 * 响应: {
 *   code, data: {
 *     nodes: [{ id, name, hash, status: 'normal'|'tampered', x, y }],
 *     edges: [{ from, to }],
 *     chainStatus: 'consistent'|'inconsistent',
 *     alert: string
 *   }
 * }
 */
export function getTopology() {
  if (useMock()) return mockTopology()
  return request.get('/audit/topology')
}

/**
 * 告警日志列表（组员 A 提供）
 * GET /api/audit/alerts
 * 响应: { code, data: [{ id, time, node, type, message }] }
 */
export function getAlerts() {
  if (useMock()) return mockAlerts()
  return request.get('/audit/alerts')
}

export function getAlertDetail(id) {
  if (useMock()) return mockAlertDetail(id)
  return request.get(`/audit/alerts/${encodeURIComponent(id)}`)
}

export function getRequestDetail(requestId) {
  if (useMock()) return mockRequestDetail(requestId)
  return request.get(`/audit/requests/${encodeURIComponent(requestId)}`)
}

export function getAuditRequests() {
  if (useMock()) return mockAuditRequests()
  return request.get('/audit/requests')
}
