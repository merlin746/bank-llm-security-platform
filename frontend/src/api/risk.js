import request from './request'
import { useMock, mockSubmitRiskReview } from '@/mock'

export function submitRiskReview(payload) {
  if (useMock()) return mockSubmitRiskReview(payload)
  return request.post('/risk/reviews', payload)
}
