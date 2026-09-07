async function unwrap(path, options = {}) {
  const resp = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const raw = await resp.text()
  let body
  try {
    body = JSON.parse(raw)
  } catch {
    body = { message: raw }
  }
  if (!resp.ok || (body && body.code !== 0 && body.code !== undefined && body.code !== 200)) {
    const message = body && (body.message || body.detail)
    if (!message) {
      throw new Error('HTTP ' + resp.status)
    }
    throw new Error(message)
  }
  return body && body.data !== undefined ? body.data : body
}

const post = (path, payload) =>
  unwrap(path, { method: 'POST', body: JSON.stringify(payload) })

export const api = {
  detect: (text) => post('/api/v1/ai/prompt/detect', { text }),
  desensitize: (text) => post('/api/v1/ai/output/desensitize', { text }),
  riskScore: (history) => post('/api/v1/ai/risk/score', { user_id: 'demo_user', history }),
  stats: () => unwrap('/api/v1/audit/stats'),
  anomalies: () => unwrap('/api/v1/audit/anomalies'),
  requests: () => unwrap('/api/v1/audit/requests'),
  requestDetail: (id) => unwrap('/api/v1/audit/requests/' + encodeURIComponent(id)),
}

export const sampleHistory = () => {
  const now = Date.now()
  const base = (mins) => new Date(now - mins * 60000).toISOString().slice(0, 19).replace('T', ' ')
  return [
    { timestamp: base(8), type: 'chat', is_attack: false, ip: '10.0.0.21' },
    { timestamp: base(6), type: 'query', is_attack: false, ip: '10.0.0.21' },
    { timestamp: base(4), type: 'chat', is_attack: false, ip: '10.0.0.22' },
    { timestamp: base(2), type: 'command', is_attack: true, ip: '10.0.0.66' },
    { timestamp: base(0), type: 'chat', is_attack: false, ip: '10.0.0.22' },
  ]
}
