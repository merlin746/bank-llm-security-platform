import { defineStore } from 'pinia'
import { login as apiLogin, logout as apiLogout, currentUser } from '@/api/auth'

const sessionVersionKey = 'chain-safe.auth-version'
const sessionVersion = '3'
let restoringSession = null

// 旧版任意账号登录曾缓存错误的角色，重新认证后再使用固定账号身份。
if (localStorage.getItem('token') && localStorage.getItem(sessionVersionKey) !== sessionVersion) {
  localStorage.removeItem('token')
  localStorage.removeItem('userInfo')
}

function readStoredUser() {
  try {
    const user = JSON.parse(localStorage.getItem('userInfo') || 'null')
    return user && typeof user === 'object' ? user : null
  } catch {
    return null
  }
}

// 用户态：登录令牌 + 当前用户信息（角色/密级），供网关页展示当前身份
export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userInfo: readStoredUser()
  }),
  getters: {
    isLogin: (state) => !!state.token && !!state.userInfo,
    username: (state) => state.userInfo?.username || '未登录',
    role: (state) => state.userInfo?.role || '-',
    dataLevel: (state) => state.userInfo?.dataLevel || '-'
  },
  actions: {
    async login(payload) {
      const res = await apiLogin(payload)
      this.token = res.data.token
      this.userInfo = res.data.user
      localStorage.setItem('token', this.token)
      localStorage.setItem('userInfo', JSON.stringify(this.userInfo))
      localStorage.setItem(sessionVersionKey, sessionVersion)
      return res
    },
    clearSession() {
      this.token = ''
      this.userInfo = null
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
      localStorage.removeItem(sessionVersionKey)
    },
    async restoreSession() {
      if (!this.token) return false
      if (restoringSession) return restoringSession
      const token = this.token
      restoringSession = (async () => {
        try {
          const res = await currentUser()
          if (this.token !== token) return this.isLogin
          const user = res.data?.user || res.data
          if (!user?.username || !Array.isArray(user.permissions)) throw new Error('登录身份无效')
          this.userInfo = user
          localStorage.setItem('userInfo', JSON.stringify(user))
          return true
        } catch {
          if (this.token === token) this.clearSession()
          return false
        } finally {
          restoringSession = null
        }
      })()
      return restoringSession
    },
    async logout() {
      const token = this.token
      // 在本地清理前发出注销，Mock也能找到需要撤销的会话。
      const pending = token ? apiLogout(token) : Promise.resolve()
      this.clearSession()
      await pending.catch(() => {})
    }
  }
})
