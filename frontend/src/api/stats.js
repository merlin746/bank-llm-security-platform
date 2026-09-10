import request from './request'
import { useMock, mockOverview, mockTrend, mockRiskDistribution, mockHighRiskUsers } from '@/mock'

/**
 * 安全态势总览 KPI（Go 网关统计聚合）
 * GET /api/stats/overview
 * 响应: { code, data: { totalRequests, blockedToday, blockRate, highRiskUsers } }
 */
export function getOverview() {
  if (useMock()) return mockOverview()
  return request.get('/stats/overview')
}

/**
 * 实时拦截量趋势
 * GET /api/stats/trend
 * 响应: { code, data: [{ time, blocked, passed }] }
 */
export function getTrend() {
  if (useMock()) return mockTrend()
  return request.get('/stats/trend')
}

/**
 * 风险类型分布
 * GET /api/stats/risk-distribution
 * 响应: { code, data: [{ type, value }] }
 */
export function getRiskDistribution() {
  if (useMock()) return mockRiskDistribution()
  return request.get('/stats/risk-distribution')
}

/**
 * 高风险用户榜单
 * GET /api/stats/high-risk-users
 * 响应: { code, data: [{ name, role, riskScore, lastAction, level }] }
 */
export function getHighRiskUsers() {
  if (useMock()) return mockHighRiskUsers()
  return request.get('/stats/high-risk-users')
}
