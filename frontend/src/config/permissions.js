// 角色控制操作，密级与业务范围控制数据；二者始终独立计算。
export const DATA_LEVELS = ['L1', 'L2', 'L3', 'L4']
export const ROLE_PERMISSIONS = Object.freeze({
  '柜员／客服': ['business.read', 'business.submit'],
  '风控审核员': ['business.read', 'dashboard.read', 'alerts.read', 'risk.review', 'simulation.run'],
  '审计人员': ['business.read', 'alerts.read', 'audit.read'],
  '系统管理员': ['admin.manage']
})

export function hasPermission(user, capability) {
  if (!user?.username || !DATA_LEVELS.includes(user.dataLevel)) return false
  const granted = ROLE_PERMISSIONS[user.role]
  if (!granted?.includes(capability)) return false
  // 服务端返回权限时使用交集，客户端不能靠附加权限扩大角色能力。
  return !Array.isArray(user.permissions) || user.permissions.includes(capability)
}

export function canAccessRecord(user, record) {
  if (!record || !hasPermission(user, 'business.read')) return false
  if (!DATA_LEVELS.includes(record.dataLevel) || DATA_LEVELS.indexOf(record.dataLevel) > DATA_LEVELS.indexOf(user.dataLevel)) return false
  const scope = user.scope
  if (!scope || !Array.isArray(scope.departments) || !scope.departments.includes(record.department)) return false
  return !scope.ownerOnly || record.owner === user.username
}

export function getDefaultRoute(user) {
  if (hasPermission(user, 'admin.manage')) return '/administration'
  if (hasPermission(user, 'audit.read')) return '/audit-topology'
  if (hasPermission(user, 'dashboard.read')) return '/dashboard'
  if (hasPermission(user, 'business.read')) return '/business'
  return '/login'
}

export function getRolePresentation(user) {
  const presentations = {
    '柜员／客服': { title: '柜员／客服工作台', description: '处理本人授权业务，查看结果与风险提示。', mode: '业务处理' },
    '风控审核员': { title: '风控审核工作台', description: '查看授权范围的安全态势，核实告警并完成风险复核。', mode: '风险复核' },
    '审计人员': { title: '审计查证工作台', description: '只读查看授权审计记录、证据链与对账结果。', mode: '只读审计' },
    '系统管理员': { title: '系统管理工作台', description: '管理账号、角色、策略与运行配置。客户数据需另行授权。', mode: '系统配置' }
  }
  const presentation = presentations[user?.role] || { title: '身份待验证', description: '请使用有效账号重新登录。', mode: '未授权' }
  const departments = user?.scope?.departments || []
  const scopeLabel = user?.role === '系统管理员' ? '平台配置 · 无客户数据授权'
    : user?.scope?.ownerOnly ? `本人业务 · ${user.dataLevel} 及以下`
      : `${departments.join('、') || '无业务范围'} · ${user?.dataLevel || '未分级'} 及以下`
  return { ...presentation, scopeLabel }
}
