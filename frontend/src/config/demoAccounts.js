// 固定演示身份；真实账号认证由后端用户存储负责。
export const demoAccounts = [
  { username: 'teller', password: 'teller', role: '柜员／客服', dataLevel: 'L1', department: '零售业务部', scope: { departments: ['零售业务部'], ownerOnly: true } },
  { username: 'reviewer', password: 'reviewer', role: '风控审核员', dataLevel: 'L3', department: '零售业务部', scope: { departments: ['零售业务部'], ownerOnly: false } },
  { username: 'auditor', password: 'auditor', role: '审计人员', dataLevel: 'L2', department: '零售业务部', scope: { departments: ['零售业务部', '信贷业务部'], ownerOnly: false } },
  { username: 'admin', password: 'admin', role: '系统管理员', dataLevel: 'L4', department: '平台运维部', scope: { departments: [], ownerOnly: false } }
]
