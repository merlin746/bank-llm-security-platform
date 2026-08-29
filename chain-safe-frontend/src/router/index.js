import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layout/MainLayout.vue'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/',
    component: MainLayout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'SecurityDashboard',
        component: () => import('@/views/SecurityDashboard.vue'),
        meta: { title: '安全态势大屏' }
      },
      {
        path: 'attack-defense',
        name: 'AttackDefense',
        component: () => import('@/views/AttackDefense.vue'),
        meta: { title: '模拟攻防测试' }
      },
      {
        path: 'audit-topology',
        name: 'AuditTopology',
        component: () => import('@/views/AuditTopology.vue'),
        meta: { title: '审计溯源对账' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  document.title = `${to.meta.title || ''} · 链安智御`
})

export default router
