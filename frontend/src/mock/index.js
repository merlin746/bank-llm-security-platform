// Mock 开关：读取 .env.development 的 VITE_USE_MOCK，为 'true' 时前端自产数据，可独立演示
export const useMock = () => import.meta.env.VITE_USE_MOCK === 'true'

export * from './data'
