import request from './request'
import { useMock, mockBusinessRequests, mockSubmitBusinessRequest } from '@/mock'

export function getBusinessRequests() {
  if (useMock()) return mockBusinessRequests()
  return request.get('/business/requests')
}

export function submitBusinessRequest(payload) {
  if (useMock()) return mockSubmitBusinessRequest(payload)
  return request.post('/business/requests', payload)
}
