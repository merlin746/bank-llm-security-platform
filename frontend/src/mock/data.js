// ============================================================
// 前端 Mock 数据：后端（Go 网关 / 组员 A 审计 / 组员 C AI）未就绪时，
// 前端可用本文件独立演示三个页面。联调时把 .env.development 的
// VITE_USE_MOCK 改为 false，即切换到真实 API。
// ============================================================

function delay(ms = 350) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function ok(data) {
  return { code: 0, data, msg: 'ok' }
}

let seq = 1000
function reqId() {
  seq += 1
  return `req-${seq}`
}

// ---------------- 登录（组员 B 业务后台） ----------------
export async function mockLogin({ username }) {
  await delay(300)
  return ok({
    token: `mock-token-${Date.now()}`,
    user: {
      username,
      role: '风控审核员',
      dataLevel: 'L3'
    }
  })
}

// ---------------- 攻防测试（Go 网关管道） ----------------
// 恶意关键词命中 → 输入风险检测拦截；否则放行
const JAILBREAK_KEYWORDS = ['忽略', '越狱', '绕过', 'system', '你现在是', 'DAN', '泄露', '忘记之前的', '无限制']

function buildStages(verdictBlocked, blockedKey, message, extra = {}) {
  const auth = { key: 'auth', name: '身份认证', owner: 'B', status: 'pass', latencyMs: 1, message: 'Token 校验通过' }
  const inputRisk = {
    key: 'input-risk',
    name: 'AI 输入风险检测',
    owner: 'C',
    status: blockedKey === 'input-risk' ? 'block' : 'pass',
    latencyMs: 12,
    message: blockedKey === 'input-risk' ? message : '未检测到注入/越狱意图',
    extra
  }
  const accessControl = {
    key: 'access-control',
    name: '权限/密级校验',
    owner: 'A',
    status: blockedKey === 'access-control' ? 'block' : 'pass',
    latencyMs: 2,
    message: blockedKey === 'access-control' ? message : '角色/密级/频次校验通过（Redis 缓存）'
  }
  const infer = {
    key: 'infer',
    name: 'LLM 推理节点',
    owner: '-',
    status: verdictBlocked ? 'skip' : 'pass',
    latencyMs: verdictBlocked ? 0 : 46,
    message: verdictBlocked ? '已被拦截，未进入推理' : '推理完成'
  }
  const outputSanitize = {
    key: 'output-sanitize',
    name: '输出脱敏与合规',
    owner: 'C',
    status: verdictBlocked ? 'skip' : 'pass',
    latencyMs: verdictBlocked ? 0 : 9,
    message: verdictBlocked ? '-' : '敏感信息已掩码，合规评分 98'
  }
  return [auth, inputRisk, accessControl, infer, outputSanitize]
}

export async function mockAttackTest({ prompt = '' }) {
  await delay()
  const hit = JAILBREAK_KEYWORDS.find((k) => prompt.toLowerCase().includes(k.toLowerCase()))
  const blocked = !!hit
  const stages = buildStages(
    blocked,
    blocked ? 'input-risk' : null,
    blocked ? `检测到越狱指令「${hit}」，意图越界` : '',
    blocked ? { riskScore: 92, riskType: 'jailbreak' } : { riskScore: 18, riskType: 'normal' }
  )
  return ok({
    requestId: reqId(),
    prompt,
    verdict: blocked ? 'block' : 'pass',
    totalLatencyMs: stages.reduce((s, x) => s + (x.latencyMs || 0), 0),
    stages
  })
}

export async function mockAccessTest({ role = '普通柜员', dataLevel = 'L3', action = '查询' }) {
  await delay()
  // 模拟越权：请求密级高于当前角色可访问密级即拦截
  const allowed = { 普通柜员: 'L2', 风控审核员: 'L3', 管理员: 'L4' }
  const canAccess = (allowed[role] || 'L1') >= dataLevel
  const stages = buildStages(
    !canAccess,
    canAccess ? null : 'access-control',
    canAccess ? '' : `角色 [${role}] 无权访问密级 [${dataLevel}] 数据`,
    { requestedLevel: dataLevel, allowedLevel: allowed[role] || 'L1' }
  )
  return ok({
    requestId: reqId(),
    role,
    dataLevel,
    action,
    verdict: canAccess ? 'pass' : 'block',
    totalLatencyMs: stages.reduce((s, x) => s + (x.latencyMs || 0), 0),
    stages
  })
}

// ---------------- 审计溯源（组员 A） ----------------
export async function mockTopology() {
  await delay(300)
  return ok({
    nodes: [
      { id: 'access', name: '访问节点', hash: '0x3a9f21c4e8b7...', status: 'normal', x: 80, y: 200 },
      { id: 'rag', name: 'RAG 检索节点', hash: '0x8c41de2a5f90...', status: 'tampered', x: 300, y: 200 },
      { id: 'inference', name: '推理节点', hash: '0x51b7e0d9c3a2...', status: 'normal', x: 520, y: 200 },
      { id: 'warehouse', name: '数仓节点', hash: '0x9e6d4f1b8a37...', status: 'normal', x: 740, y: 200 }
    ],
    edges: [
      { from: 'access', to: 'rag' },
      { from: 'rag', to: 'inference' },
      { from: 'inference', to: 'warehouse' }
    ],
    chainStatus: 'inconsistent',
    alert: 'RAG 检索节点 Hash 与链上记录不一致，疑似被篡改'
  })
}

export async function mockAlerts() {
  await delay(200)
  return ok([
    { id: 1, time: '2026-08-29 14:32:11', node: 'RAG 检索节点', type: 'hash-mismatch', message: '节点 Hash 与链上不一致' },
    { id: 2, time: '2026-08-29 13:05:44', node: '访问节点', type: 'privilege', message: '越权访问密级 L4 数据被拦截' },
    { id: 3, time: '2026-08-29 11:20:03', node: '推理节点', type: 'jailbreak', message: '检测到 Prompt 越狱意图' }
  ])
}

// ---------------- 安全态势（Go 网关统计） ----------------
export async function mockOverview() {
  await delay(200)
  return ok({
    totalRequests: 12894,
    blockedToday: 321,
    blockRate: 2.49,
    highRiskUsers: 7
  })
}

export async function mockTrend() {
  await delay(200)
  const hours = ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
  return ok(hours.map((time, i) => ({ time, blocked: [40, 22, 68, 95, 71, 52, 31][i], passed: [800, 620, 1500, 2300, 1900, 1400, 900][i] })))
}

export async function mockRiskDistribution() {
  await delay(200)
  return ok([
    { type: 'Prompt 注入', value: 128 },
    { type: '越权访问', value: 96 },
    { type: '敏感数据泄露', value: 54 },
    { type: '违规承诺', value: 28 },
    { type: '频次异常', value: 15 }
  ])
}

export async function mockHighRiskUsers() {
  await delay(200)
  return ok([
    { name: 'u_200731', role: '普通柜员', riskScore: 96, lastAction: '越权查询密级 L4', level: '高' },
    { name: 'u_100288', role: '风控审核员', riskScore: 87, lastAction: '高频 Prompt 注入尝试', level: '高' },
    { name: 'u_301445', role: '客服坐席', riskScore: 74, lastAction: '导出敏感字段', level: '中' },
    { name: 'u_400912', role: '普通柜员', riskScore: 68, lastAction: '异常时段访问', level: '中' }
  ])
}

// ---------------- 业务后台 CRUD（组员 B） ----------------
export async function mockUsers() {
  await delay(200)
  return ok([
    { id: 1, username: 'admin', role: '管理员', dataLevel: 'L4' },
    { id: 2, username: 'risk_01', role: '风控审核员', dataLevel: 'L3' },
    { id: 3, username: 'teller_01', role: '普通柜员', dataLevel: 'L2' }
  ])
}

export async function mockRoles() {
  await delay(200)
  return ok(['管理员', '风控审核员', '普通柜员', '客服坐席'])
}

export async function mockDataLevels() {
  await delay(200)
  return ok([
    { level: 'L1', desc: '公开数据', fields: '无' },
    { level: 'L2', desc: '内部数据', fields: '基础客户信息' },
    { level: 'L3', desc: '敏感数据', fields: '账户余额、交易明细' },
    { level: 'L4', desc: '高度敏感', fields: '身份证、卡号、征信' }
  ])
}
