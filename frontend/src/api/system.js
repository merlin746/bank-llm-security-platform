import request from './request'
import { useMock, mockSystemSettings, mockUpdateSystemSettings } from '@/mock'

export function getSystemSettings() {
  if (useMock()) return mockSystemSettings()
  return request.get('/system/settings')
}

export function updateSystemSettings(data) {
  if (useMock()) return mockUpdateSystemSettings(data)
  return request.put('/system/settings', data)
}
