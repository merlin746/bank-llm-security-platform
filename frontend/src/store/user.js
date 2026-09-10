import { defineStore } from 'pinia'
import { login as apiLogin } from '@/api/auth'

// 用户态：登录令牌 + 当前用户信息（角色/密级），供网关页展示当前身份
export const useUserStore = defineStore('user', {
  state: () => ({
    token: localStorage.getItem('token') || '',
    userInfo: JSON.parse(localStorage.getItem('userInfo') || 'null')
  }),
  getters: {
    isLogin: (state) => !!state.token,
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
      return res
    },
    logout() {
      this.token = ''
      this.userInfo = null
      localStorage.removeItem('token')
      localStorage.removeItem('userInfo')
    }
  }
})
