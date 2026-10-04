// ============================================================
// 前端 Mock 数据：后端（Go 网关 / 组员 A 审计 / 组员 C AI）未就绪时，
// 前端可用本文件独立演示三个页面。联调时把 .env.development 的
// VITE_USE_MOCK 改为 false，即切换到真实 API。
// ============================================================

import { demoAccounts } from '../config/demoAccounts.js'
import { ROLE_PERMISSIONS, DATA_LEVELS, hasPermission, canAccessRecord } from '../config/permissions.js'

const ACCESS_KEY = 'chain-safe.mock-access.v4'
const HISTORY_KEY = 'chain-safe.mock-history.v3'
const HISTORY_LIMIT = 200

function readStorage(key, fallback) {
  try {
    return JSON.parse(globalThis.sessionStorage?.getItem(key) || 'null') || fallback()
  } catch {
    return fallback()
  }
}

const access = readStorage(ACCESS_KEY, () => ({
  accounts: demoAccounts.map((account, index) => ({ ...account, id: index + 1, revision: 1 })),
  sessions: {},
  nextAccountId: demoAccounts.length + 1,
  settings: { policy: { allowGatewayTests: true }, runtime: { aiTimeoutMs: 4000 } }
}))
const history = readStorage(HISTORY_KEY, () => ({ requests: [], alerts: [] }))

function persist(key, value) {
  try {
    globalThis.sessionStorage?.setItem(key, JSON.stringify(value))
  } catch {
    // 存储关闭时仍在当前页面内保留已签发会话和关联记录。
  }
}

function fail(code, message) {
  const error = new Error(message)
  error.code = code
  error.status = code
  error.response = { status: code, data: { code, msg: message } }
  throw error
}

function publicUser(account) {
  const { id, username, role, dataLevel, department, scope } = account
  return { id, username, role, dataLevel, department, permissions: [...(ROLE_PERMISSIONS[role] || [])], scope }
}

async function authorize(capability, ms = 200) {
  let token
  try {
    token = globalThis.localStorage?.getItem('token')
  } catch {
    fail(401, '请先登录')
  }
  const session = access.sessions[token]
  const account = session && access.accounts.find((item) => item.id === session.accountId && item.revision === session.revision)
  if (!account) fail(401, '登录已失效，请重新登录')
  const user = publicUser(account)
  if (!DATA_LEVELS.includes(user.dataLevel) || !ROLE_PERMISSIONS[user.role]) fail(403, '当前身份未获得有效授权')
  if (capability && !hasPermission(user, capability)) fail(403, '当前账号无权执行此操作')
  await delay(ms)
  if (token !== globalThis.localStorage?.getItem('token') || !access.sessions[token]) fail(401, '会话已切换，请重新加载当前页面')
  return user
}

function canReadRecord(user, record) {
  return canAccessRecord(user, record)
}

function visibleRequests(user) {
  return [...history.requests, ...SEED_REQUESTS].filter((item) => canReadRecord(user, item))
}

async function gatewayUser() {
  const user = await authorize('simulation.run', 350)
  if (!access.settings.policy.allowGatewayTests) fail(403, '系统策略已暂停攻防测试')
  if (!canReadRecord(user, { owner: user.username, department: user.scope.departments[0], dataLevel: user.dataLevel })) fail(403, '当前账号没有可用于模拟测试的业务授权范围')
  return user
}

function delay(ms = 350) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function ok(data) {
  return { code: 0, data: JSON.parse(JSON.stringify(data)), msg: 'ok' }
}

let seq = 1000
function reqId() {
  seq += 1
  return `req-mock-${globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(36).slice(2)}-${seq}`}`
}

function saveRequest(result, kind, user, alert = null) {
  const time = new Date().toLocaleString('sv-SE', { timeZone: 'Asia/Shanghai' })
  const ownership = { username: user.username, owner: user.username, department: user.scope.departments[0] || user.department, dataLevel: result.recordLevel || result.dataLevel || user.dataLevel }
  const alertIds = []
  if (alert) {
    const id = Math.max(Date.now(), ...history.alerts.map((item) => item.id + 1))
    history.alerts.unshift({ ...alert, ...ownership, id, time, requestId: result.requestId, severity: 'high', status: 'blocked' })
    alertIds.push(id)
  }
  const request = { ...result, ...ownership, role: user.role, kind, time, alertIds }
  delete request.recordLevel
  history.requests.unshift(request)
  history.requests = history.requests.slice(0, HISTORY_LIMIT)
  const keptRequests = new Set(history.requests.map((item) => item.requestId))
  history.alerts = history.alerts.filter((item) => keptRequests.has(item.requestId))
  persist(HISTORY_KEY, history)
  return ok(request)
}

function mockDetection() {
  return { source: 'mock', mode: 'mock', degraded: false, reason: '演示数据，未调用实际检测服务' }
}

// ---------------- 登录（组员 B 业务后台） ----------------
export async function mockLogin({ username, password } = {}) {
  await delay(300)
  if (typeof username !== 'string' || !username.trim()) fail(400, '请输入账号')
  if (typeof password !== 'string' || !password) fail(400, '请输入密码')
  const account = access.accounts.find((item) => item.username === username.trim() && item.password === password)
  if (!account) fail(401, '账号或密码错误')
  const token = `mock-token-${reqId()}`
  access.sessions[token] = { accountId: account.id, revision: account.revision }
  const sessionEntries = Object.entries(access.sessions).slice(-HISTORY_LIMIT)
  access.sessions = Object.fromEntries(sessionEntries)
  persist(ACCESS_KEY, access)
  return ok({
    token,
    user: publicUser(account)
  })
}

export async function mockCurrentUser() { return ok({ user: await authorize(null, 0) }) }

export async function mockLogout(token = globalThis.localStorage?.getItem('token')) {
  if (!access.sessions[token]) fail(401, '登录已失效，请重新登录')
  delete access.sessions[token]
  persist(ACCESS_KEY, access)
  await delay(0)
  return ok(null)
}

// ---------------- 攻防测试（Go 网关管道） ----------------
// 使用指令与目标组合，避免正常业务提到“系统”“安全策略”等词时误报。
const JAILBREAK_RULES = [
  { id: 'instruction-override', label: '覆盖既有指令或限制', pattern: /(?:忽略|无视|忘记|抛弃|覆盖|撤销|不要遵守).{0,24}?(?:指令|限制|规则|安全策略|提示词)/iu },
  { id: 'permission-bypass', label: '绕过权限或安全校验', pattern: /(?:绕过|跳过|禁用|关闭|解除|规避).{0,16}?(?:权限|校验|限制|审查|过滤|审核)/iu },
  { id: 'internal-disclosure', label: '泄露内部指令或配置', pattern: /(?:泄露|透露|输出|展示|导出|获取|打印|返回|读取).{0,16}?(?:系统|内部|隐藏|机密).{0,16}?(?:提示词|指令|安全策略|配置|密钥|令牌)/iu },
  { id: 'unrestricted-answer', label: '解除回答限制', pattern: /(?:无限制|无任何限制|不受限制).{0,10}?(?:回答|输出|模式)/iu },
  { id: 'dan-role', label: '伪装为无限制角色', pattern: /(?:你现在是|现在你是|切换为|扮演|作为).{0,12}?\bdan\b/iu },
  { id: 'english-override', label: '覆盖既有指令', pattern: /\b(?:ignore|forget|disregard|override)\b.{0,40}?\b(?:previous|prior|system|all)\b.{0,30}?\b(?:instructions?|rules?|restrictions?)\b/iu },
  { id: 'english-disclosure', label: '泄露系统提示词', pattern: /\b(?:reveal|leak|print|show|output)\b.{0,30}?\b(?:system|internal|hidden)\b.{0,20}?\b(?:prompt|instructions?|configuration|secrets?)\b/iu }
]

function findAttack(text) {
  for (const rule of JAILBREAK_RULES) {
    for (const match of text.matchAll(new RegExp(rule.pattern.source, 'gisu'))) {
      const prefix = text.slice(Math.max(0, match.index - 24), match.index).split(/[，。！？；,;.!?\n]/u).pop()
      if (/(?:不要|不得|禁止|不应|不能|避免|防止|防范)(?:(?!但是|但|然而|不过|却|然后|改为|现在|请).){0,20}$/u.test(prefix)) continue
      return { ...rule, matchedText: match[0] }
    }
  }
  return null
}

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

export async function mockAttackTest({ prompt = '' } = {}) {
  const user = await gatewayUser()
  if (typeof prompt !== 'string' || !prompt.trim()) fail(400, '请输入待检测的 Prompt')
  const normalized = prompt.normalize('NFKC').replace(/[\u200B-\u200D\uFEFF]/gu, '')
  const hit = findAttack(normalized)
  const blocked = !!hit
  const matchedText = hit?.matchedText || ''
  const stages = buildStages(
    blocked,
    blocked ? 'input-risk' : null,
    blocked ? `检测到${hit.label}：「${matchedText}」` : '',
    blocked ? { riskScore: 92, riskType: 'jailbreak', ruleId: hit.id, matchedText } : { riskScore: 18, riskType: 'normal' }
  )
  const result = {
    requestId: reqId(),
    prompt,
    verdict: blocked ? 'block' : 'pass',
    detection: mockDetection(),
    totalLatencyMs: stages.reduce((s, x) => s + (x.latencyMs || 0), 0),
    stages
  }
  return saveRequest(result, 'attack', user, blocked ? {
    node: 'AI 输入风险检测', type: 'jailbreak', message: stages[1].message,
    evidence: `演示规则 ${hit.id} 命中「${matchedText}」，风险评分 92。`,
    recommendation: '检查关联请求中的完整 Prompt，移除覆盖指令、越权或泄露要求后重新测试。'
  } : null)
}

export async function mockAccessTest({ role, dataLevel = 'L3', action = '查询' } = {}) {
  const user = await gatewayUser()
  if (role && role !== user.role) fail(403, '只能使用当前登录角色进行测试')
  if (!/^L[1-4]$/u.test(dataLevel)) fail(400, '请求密级必须为 L1 至 L4')
  role = user.role
  // 实际密级取已认证账户，客户端不能通过 role 提升权限。
  const canAccess = Number(user.dataLevel.slice(1)) >= Number(dataLevel.slice(1))
  const stages = buildStages(
    !canAccess,
    canAccess ? null : 'access-control',
    canAccess ? '' : `角色 [${role}] 无权访问密级 [${dataLevel}] 数据`,
    { requestedLevel: dataLevel, allowedLevel: user.dataLevel }
  )
  const result = {
    requestId: reqId(),
    role,
    dataLevel,
    requestedLevel: dataLevel,
    recordLevel: user.dataLevel,
    action,
    verdict: canAccess ? 'pass' : 'block',
    detection: mockDetection(),
    totalLatencyMs: stages.reduce((s, x) => s + (x.latencyMs || 0), 0),
    stages
  }
  return saveRequest(result, 'access', user, canAccess ? null : {
    node: '权限/密级校验', type: 'privilege', message: stages[2].message,
    evidence: `角色 ${role} 的当前授权密级为 ${user.dataLevel}，本次请求${action} ${dataLevel} 数据。`,
    recommendation: '使用具备相应权限的角色或降低请求密级；需要授权时按业务审批流程申请。'
  })
}

// ---------------- 审计溯源（组员 A） ----------------
const SEED_REQUESTS = [
  { requestId: 'biz-demo-own', owner: 'teller', department: '零售业务部', dataLevel: 'L1', businessType: '产品与服务咨询', prompt: '请问如何办理银行卡挂失', result: '演示结果：请通过银行官方渠道办理挂失并核验身份。' },
  { requestId: 'biz-demo-retail', owner: 'retail-agent', department: '零售业务部', dataLevel: 'L1', businessType: '零售业务处理', prompt: '查询开户办理流程', result: '演示结果：已完成零售业务流程查询。' },
  { requestId: 'biz-demo-credit', owner: 'credit-agent', department: '信贷业务部', dataLevel: 'L2', businessType: '信贷内部流程', prompt: '核对信贷内部审批流程', result: '演示结果：仅展示授权范围内的内部审批流程。' },
  { requestId: 'biz-demo-sensitive', owner: 'reviewer', department: '零售业务部', dataLevel: 'L3', businessType: '敏感业务复核', prompt: '核实授权范围内的敏感业务', result: '演示结果：已完成敏感业务风险核实。' }
].map((request) => ({ ...request, username: request.owner, time: '2026-10-02 14:00:00', kind: 'business', verdict: 'pass', status: 'completed', detection: mockDetection(), alertIds: [], stages: [] }))

export async function mockBusinessRequests() {
  const user = await authorize('business.read')
  return ok(visibleRequests(user))
}

export async function mockAuditRequests() {
  const user = await authorize('audit.read')
  return ok(visibleRequests(user))
}

export async function mockSubmitBusinessRequest({ prompt = '', dataLevel = 'L1', businessType = '产品与服务咨询', action, role, department } = {}) {
  const user = await authorize('business.submit', 350)
  if (typeof prompt !== 'string' || !prompt.trim()) fail(400, '请输入需要处理的业务内容')
  if (role && role !== user.role) fail(403, '不能伪造业务角色')
  const recordDepartment = department || user.scope.departments[0]
  if (!canReadRecord(user, { dataLevel, department: recordDepartment, owner: user.username })) fail(403, '业务密级或部门超出当前授权范围')
  const normalized = prompt.normalize('NFKC').replace(/[\u200B-\u200D\uFEFF]/gu, '')
  const hit = findAttack(normalized)
  const blocked = !!hit
  const stages = buildStages(blocked, blocked ? 'input-risk' : null, blocked ? `检测到${hit.label}：「${hit.matchedText}」` : '', blocked ? { riskScore: 92, riskType: 'jailbreak', ruleId: hit.id, matchedText: hit.matchedText } : { riskScore: 18, riskType: 'normal' })
  const result = {
    requestId: reqId(), prompt: prompt.trim(), dataLevel, businessType, action: action || businessType,
    verdict: blocked ? 'block' : 'pass', status: blocked ? 'blocked' : 'completed',
    result: blocked ? '业务请求存在风险，已在处理前阻断。' : '业务请求已完成安全检查，请通过银行官方业务流程继续办理。',
    riskTip: blocked ? '检测到覆盖指令、越权或泄露要求，请核实业务内容后重新提交。' : '未发现攻击特征，结果仅限当前授权范围。',
    detection: mockDetection(), stages, totalLatencyMs: stages.reduce((sum, stage) => sum + stage.latencyMs, 0)
  }
  return saveRequest(result, 'business', user, blocked ? {
    node: 'AI 输入风险检测', type: 'jailbreak', message: stages[1].message,
    evidence: `演示规则 ${hit.id} 命中「${hit.matchedText}」，风险评分 92。`, recommendation: '复核关联业务内容，移除越权或泄露要求后重新提交。'
  } : null)
}

export async function mockTopology() {
  const user = await authorize('audit.read', 300)
  if (!canReadRecord(user, SEED_ALERTS[0])) return ok({ nodes: [], edges: [], chainStatus: 'unavailable', alert: '当前授权范围内暂无完整性对账证据' })
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

const SEED_ALERTS = [
  { id: 1, time: '2026-08-29 14:32:11', node: 'RAG 检索节点', type: 'hash-mismatch', message: '节点 Hash 与链上不一致', severity: 'high', status: 'open', requestId: '', evidence: '历史演示告警，未保存原始 Hash 和请求记录，无法关联请求。', recommendation: '核对 RAG 节点的数据完整性及链上记录。' },
  { id: 2, time: '2026-08-29 13:05:44', node: '访问节点', type: 'privilege', message: '越权访问密级 L4 数据被拦截', severity: 'high', status: 'blocked', requestId: '', evidence: '历史演示告警，未保存角色、权限依据和原始请求记录，无法关联请求。', recommendation: '复核访问角色及密级授权。' },
  { id: 3, time: '2026-08-29 11:20:03', node: '推理节点', type: 'jailbreak', message: '检测到 Prompt 越狱意图', severity: 'high', status: 'blocked', requestId: '', evidence: '历史演示告警，未保存原始 Prompt 和请求记录，无法关联请求。', recommendation: '通过攻防测试生成新的检测请求以查看完整链路。' }
].map((alert, index) => ({ ...alert, username: '', owner: '', department: '零售业务部', dataLevel: index === 2 ? 'L1' : 'L2' }))

function allAlerts() {
  return [...history.alerts, ...SEED_ALERTS.map((alert) => ({ ...alert, ...history.seedReviews?.[alert.id] }))]
}

function visibleAlerts(user) {
  return allAlerts().filter((item) => canReadRecord(user, item))
}

function findVisibleAlert(id, user) {
  const alert = allAlerts().find((item) => String(item.id) === String(id))
  if (!alert) fail(404, '告警不存在或已超过演示记录保留上限')
  if (!canReadRecord(user, alert)) fail(403, '当前账号无权读取此告警')
  return alert
}

export async function mockAlerts() {
  const user = await authorize('alerts.read')
  return ok(visibleAlerts(user))
}

export async function mockAlertDetail(id) {
  const user = await authorize('alerts.read')
  return ok(findVisibleAlert(id, user))
}

export async function mockRequestDetail(requestId) {
  const user = await authorize('business.read')
  const request = [...history.requests, ...SEED_REQUESTS].find((item) => item.requestId === requestId)
  if (!request) fail(404, '请求不存在或已超过演示记录保留上限')
  if (!canReadRecord(user, request)) fail(403, '当前账号无权读取此请求')
  return ok(request)
}

export async function mockWorkspaceRequests() {
  const user = await authorize('business.read')
  return ok(visibleRequests(user))
}

export async function mockSubmitRiskReview({ alertId, decision, note } = {}) {
  const user = await authorize('risk.review')
  if (!['confirmed', 'dismissed'].includes(decision)) fail(400, '请选择风险确认或误报排除')
  if (typeof note !== 'string' || !note.trim() || note.trim().length > 1000) fail(400, '请填写不超过 1000 字的复核说明')
  const alert = findVisibleAlert(alertId, user)
  alert.status = 'reviewed'
  alert.review = { decision, note: note.trim(), reviewer: user.username, time: new Date().toLocaleString('sv-SE', { timeZone: 'Asia/Shanghai' }) }
  if (SEED_ALERTS.some((item) => item.id === alert.id)) {
    history.seedReviews ||= {}
    history.seedReviews[alert.id] = { status: alert.status, review: alert.review }
  }
  persist(HISTORY_KEY, history)
  return ok(alert)
}

// ---------------- 安全态势（Go 网关统计） ----------------
export async function mockOverview() {
  const user = await authorize('dashboard.read')
  const requests = visibleRequests(user)
  const blocked = requests.filter((item) => item.verdict === 'block')
  const today = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
  return ok({
    totalRequests: requests.length,
    blockedToday: blocked.filter((item) => item.time.startsWith(today)).length,
    blockRate: requests.length ? Math.round(blocked.length / requests.length * 10000) / 100 : 0,
    highRiskUsers: new Set(blocked.map((item) => item.username)).size
  })
}

export async function mockTrend() {
  const user = await authorize('dashboard.read')
  const hours = ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00']
  const trend = hours.map((time) => ({ time, blocked: 0, passed: 0 }))
  const today = new Date().toLocaleDateString('sv-SE', { timeZone: 'Asia/Shanghai' })
  for (const item of visibleRequests(user).filter((request) => request.time.startsWith(today))) {
    const index = Math.floor(Number(item.time.slice(11, 13)) / 4)
    trend[index][item.verdict === 'block' ? 'blocked' : 'passed'] += 1
  }
  return ok(trend)
}

export async function mockRiskDistribution() {
  const user = await authorize('dashboard.read')
  const counts = new Map()
  const labels = { jailbreak: 'Prompt 注入', privilege: '越权访问', 'hash-mismatch': '完整性异常' }
  for (const alert of visibleAlerts(user)) {
    const type = labels[alert.type] || alert.type
    counts.set(type, (counts.get(type) || 0) + 1)
  }
  return ok([...counts].map(([type, value]) => ({ type, value })))
}

export async function mockHighRiskUsers() {
  const user = await authorize('dashboard.read')
  const users = new Map()
  for (const request of visibleRequests(user).filter((item) => item.verdict === 'block')) {
    const riskScore = request.stages.find((stage) => stage.key === 'input-risk')?.extra?.riskScore || 90
    if (users.has(request.username)) continue
    users.set(request.username, {
      name: request.username, role: request.role, riskScore,
      lastAction: request.kind === 'access' ? `${request.action}密级 ${request.dataLevel} 数据` : 'Prompt 注入测试被拦截', level: '高'
    })
  }
  return ok([...users.values()].sort((a, b) => b.riskScore - a.riskScore))
}

// ---------------- 业务后台 CRUD（组员 B） ----------------
export async function mockUsers() {
  await authorize('admin.manage')
  return ok(access.accounts.map(publicUser))
}

export async function mockRoles() {
  await authorize('admin.manage')
  const roleProfiles = demoAccounts
  return ok(roleProfiles.map((profile) => ({
    name: profile.role,
    maxAccessLevel: 'L4',
    defaultDataLevel: profile.dataLevel,
    department: profile.department,
    permissions: ROLE_PERMISSIONS[profile.role],
    scope: profile.scope,
    chainRoleOrdinal: { '柜员／客服': 2, '风控审核员': 3, '审计人员': 1, '系统管理员': 4 }[profile.role]
  })))
}

export async function mockDataLevels() {
  await authorize('admin.manage')
  return ok([
    { level: 'L1', rank: 1, desc: '公开数据', fields: '无' },
    { level: 'L2', rank: 2, desc: '内部数据', fields: '基础客户信息' },
    { level: 'L3', rank: 3, desc: '敏感数据', fields: '账户余额、交易明细' },
    { level: 'L4', rank: 4, desc: '高度敏感', fields: '身份证、卡号、征信' }
  ])
}

function findAccount(id) {
  const account = access.accounts.find((item) => String(item.id) === String(id))
  if (!account) fail(404, '账号不存在')
  return account
}

function accountAuthorization(payload, current = null) {
  const role = payload.role || current?.role
  const profile = demoAccounts.find((item) => item.role === role)
  if (!profile) fail(400, '请选择有效角色')
  const dataLevel = payload.dataLevel || current?.dataLevel || profile.dataLevel
  if (!DATA_LEVELS.includes(dataLevel)) fail(400, '密级必须为 L1 至 L4')
  const isAdmin = role === '系统管理员'
  const department = payload.department || current?.department || (isAdmin ? '平台运维部' : profile.scope.departments[0])
  if (isAdmin ? department !== '平台运维部' : !['零售业务部', '信贷业务部'].includes(department)) fail(400, '请选择有效业务部门')
  const allowedDepartments = isAdmin ? [] : role === '审计人员' ? ['零售业务部', '信贷业务部'] : [department]
  const scope = payload.scope || (current?.role === role && current?.department === department ? current.scope : null) || { departments: allowedDepartments, ownerOnly: role === '柜员／客服' }
  if (!Array.isArray(scope.departments) || typeof scope.ownerOnly !== 'boolean' || scope.departments.some((item) => !allowedDepartments.includes(item))) fail(400, '授权范围超出该角色的职责范围')
  if (role === '柜员／客服' && (!scope.ownerOnly || scope.departments.length !== 1)) fail(400, '柜员／客服仅能访问本部门本人业务')
  if (isAdmin && (scope.departments.length || scope.ownerOnly)) fail(400, '系统管理员不能配置客户业务范围')
  return { role, dataLevel, department, scope: { departments: [...new Set(scope.departments)], ownerOnly: scope.ownerOnly } }
}

function invalidateAccountSessions(id) {
  for (const [token, session] of Object.entries(access.sessions)) {
    if (session.accountId === id) delete access.sessions[token]
  }
}

export async function mockUserDetail(id) {
  await authorize('admin.manage')
  return ok(publicUser(findAccount(id)))
}

export async function mockCreateUser(payload = {}) {
  await authorize('admin.manage')
  if (typeof payload.username !== 'string' || !/^[A-Za-z][A-Za-z_-]*$/u.test(payload.username.trim())) fail(400, '账号请使用英文名称，可包含下划线或连字符')
  const username = payload.username.trim()
  if (access.accounts.some((account) => account.username === username)) fail(409, '账号已存在')
  if (typeof payload.password !== 'string' || !payload.password.trim()) fail(400, '请设置登录密码')
  const account = { ...accountAuthorization(payload), id: access.nextAccountId++, username, password: payload.password, revision: 1 }
  access.accounts.push(account)
  persist(ACCESS_KEY, access)
  return ok(publicUser(account))
}

export async function mockUpdateUser(id, payload = {}) {
  await authorize('admin.manage')
  const account = findAccount(id)
  if (payload.username && payload.username !== account.username) fail(400, '账号名称不可修改')
  if (account.username === 'admin' && ((payload.role && payload.role !== '系统管理员') || (payload.dataLevel && payload.dataLevel !== 'L4') || (payload.department && payload.department !== '平台运维部') || (payload.scope && (payload.scope.departments?.length || payload.scope.ownerOnly)))) fail(403, '内置系统管理员的职责与授权范围不可变更')
  const fields = accountAuthorization(payload, account)
  if (payload.password !== undefined && (typeof payload.password !== 'string' || !payload.password.trim())) fail(400, '请设置有效登录密码')
  Object.assign(account, fields, payload.password === undefined ? {} : { password: payload.password })
  account.revision += 1
  invalidateAccountSessions(account.id)
  persist(ACCESS_KEY, access)
  return ok(publicUser(account))
}

export async function mockDeleteUser(id) {
  await authorize('admin.manage')
  const account = findAccount(id)
  if (account.username === 'admin') fail(403, '内置系统管理员账号不能删除')
  access.accounts = access.accounts.filter((item) => item.id !== account.id)
  invalidateAccountSessions(account.id)
  persist(ACCESS_KEY, access)
  return ok({ id: account.id, deleted: true })
}

export async function mockSystemSettings() {
  await authorize('admin.manage')
  return ok(access.settings)
}

export async function mockUpdateSystemSettings(payload = {}) {
  const user = await authorize('admin.manage')
  if (!payload.policy && !payload.runtime) fail(400, '请提供策略或运行配置')
  if (payload.policy && typeof payload.policy.allowGatewayTests !== 'boolean') fail(400, '网关测试开关必须为布尔值')
  if (payload.runtime && (!Number.isInteger(payload.runtime.aiTimeoutMs) || payload.runtime.aiTimeoutMs < 500 || payload.runtime.aiTimeoutMs > 8000)) fail(400, 'AI 等待时间需为 500 至 8000 毫秒的整数')
  if (payload.policy) access.settings.policy = { allowGatewayTests: payload.policy.allowGatewayTests }
  if (payload.runtime) access.settings.runtime = { aiTimeoutMs: payload.runtime.aiTimeoutMs }
  access.settings.updatedBy = user.username
  access.settings.updatedAt = new Date().toLocaleString('sv-SE', { timeZone: 'Asia/Shanghai' })
  persist(ACCESS_KEY, access)
  return ok(access.settings)
}
