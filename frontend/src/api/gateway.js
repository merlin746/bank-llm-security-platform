import request from './request'
import { useMock, mockAttackTest, mockAccessTest } from '@/mock'

/**
 * 攻防测试 —— Prompt 注入/越狱攻击模拟
 * 由 Go 安全网关转发：组员 C 的 AI 输入检测 + 组员 A 的 Redis 权限校验。
 * POST /api/gateway/attack-test
 * 请求体: { prompt: string }
 * 响应: {
 *   code, data: {
 *     requestId, prompt, verdict: 'block'|'pass', totalLatencyMs,
 *     stages: [{ key, name, owner, status: 'pass'|'block'|'skip', latencyMs, message, extra }]
 *   }
 * }
 */
export function attackTest(payload) {
  if (useMock()) return mockAttackTest(payload)
  return request.post('/gateway/attack-test', payload)
}

/**
 * 攻防测试 —— 越权访问模拟
 * POST /api/gateway/access-test
 * 请求体: { role, dataLevel, action }
 * 响应:   同 attack-test 的管道结构
 */
export function accessTest(payload) {
  if (useMock()) return mockAccessTest(payload)
  return request.post('/gateway/access-test', payload)
}
