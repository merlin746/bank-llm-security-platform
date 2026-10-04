import test from 'node:test'
import assert from 'node:assert/strict'
import { demoAccounts } from '../src/config/demoAccounts.js'
import { ROLE_PERMISSIONS, hasPermission, canAccessRecord, getDefaultRoute } from '../src/config/permissions.js'

const expected = {
  teller: ['business.read', 'business.submit'],
  reviewer: ['business.read', 'dashboard.read', 'alerts.read', 'risk.review', 'simulation.run'],
  auditor: ['business.read', 'alerts.read', 'audit.read'],
  admin: ['admin.manage']
}
const users = Object.fromEntries(demoAccounts.map(user => [user.username, { ...user, permissions: ROLE_PERMISSIONS[user.role] }]))
test('four English accounts retain distinct roles and all four default data levels', () => {
  assert.deepEqual(Object.keys(users), Object.keys(expected))
  assert.equal(new Set(Object.values(users).map(user => user.dataLevel)).size, 4)
  assert.equal(users.reviewer.dataLevel, 'L3')
  for (const user of Object.values(users)) {
    for (const permission of new Set(Object.values(expected).flat())) {
      assert.equal(hasPermission(user, permission), expected[user.username].includes(permission), `${user.username}: ${permission}`)
    }
  }
})
test('each account starts at its authorized workbench', () => {
  assert.deepEqual(Object.values(users).map(getDefaultRoute), ['/business', '/dashboard', '/audit-topology', '/administration'])
})
test('record visibility requires classification, department and owner together', () => {
  const record = { owner: 'teller', department: '零售业务部', dataLevel: 'L1' }
  assert.equal(canAccessRecord(users.teller, record), true)
  assert.equal(canAccessRecord(users.teller, { ...record, owner: 'other' }), false)
  assert.equal(canAccessRecord(users.teller, { ...record, department: '信贷业务部' }), false)
  assert.equal(canAccessRecord(users.teller, { ...record, dataLevel: 'L2' }), false)
  assert.equal(canAccessRecord(users.reviewer, { ...record, owner: 'other', dataLevel: 'L3' }), true)
  assert.equal(canAccessRecord(users.reviewer, { ...record, dataLevel: 'L4' }), false)
  assert.equal(canAccessRecord(users.auditor, { ...record, department: '信贷业务部', dataLevel: 'L2' }), true)
  assert.equal(canAccessRecord(users.auditor, { ...record, dataLevel: 'L3' }), false)
  assert.equal(canAccessRecord(users.admin, record), false)
})
test('an extra client permission cannot grant administrator customer data', () => {
  const modified = { ...users.admin, permissions: ['admin.manage', 'business.read', 'simulation.run'], scope: { departments: ['零售业务部'], ownerOnly: false } }
  assert.equal(hasPermission(modified, 'simulation.run'), false)
  assert.equal(canAccessRecord(modified, { owner: 'teller', department: '零售业务部', dataLevel: 'L1' }), false)
})
test('revoked or malformed identity fails closed', () => {
  assert.equal(hasPermission({ ...users.reviewer, permissions: [] }, 'alerts.read'), false)
  assert.equal(hasPermission({ ...users.reviewer, dataLevel: 'L9' }, 'alerts.read'), false)
  assert.equal(canAccessRecord(users.reviewer, { owner: 'teller', dataLevel: 'L1' }), false)
  assert.equal(getDefaultRoute(null), '/login')
})
