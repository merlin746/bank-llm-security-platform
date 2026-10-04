import { createRouter, createWebHistory } from 'vue-router'
import MainLayout from '@/layout/MainLayout.vue'
import { useUserStore } from '@/store/user'
import { getDefaultRoute, hasPermission } from '@/config/permissions'

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
    meta: { requiresAuth: true },
    redirect: () => getDefaultRoute(useUserStore().userInfo),
    children: [
      {
        path: 'business', name: 'BusinessWorkbench',
        component: () => import('@/views/BusinessWorkbench.vue'),
        meta: { title: '业务工作台', permission: 'business.read' }
      },
      {
        path: 'dashboard',
        name: 'SecurityDashboard',
        component: () => import('@/views/SecurityDashboard.vue'),
        meta: { title: '授权安全态势', permission: 'dashboard.read' }
      },
      {
        path: 'attack-defense',
        name: 'AttackDefense',
        component: () => import('@/views/AttackDefense.vue'),
        meta: { title: '模拟攻防测试', permission: 'simulation.run' }
      },
      {
        path: 'audit-topology',
        name: 'AuditTopology',
        component: () => import('@/views/AuditTopology.vue'),
        meta: { title: '审计溯源对账', permission: 'audit.read' }
      },
      {
        path: 'risk-review', name: 'RiskReview',
        component: () => import('@/views/RiskReview.vue'),
        meta: { title: '告警与风险复核', permission: 'risk.review' }
      },
      {
        path: 'administration', name: 'Administration',
        component: () => import('@/views/SystemAdmin.vue'),
        meta: { title: '系统管理', permission: 'admin.manage' }
      },
      {
        path: 'my-access', name: 'AccessOverview',
        component: () => import('@/views/AccessOverview.vue'),
        meta: { title: '我的权限' }
      },
      {
        path: 'access-denied', name: 'AccessDenied',
        component: () => import('@/views/AccessDenied.vue'),
        meta: { title: '访问受限' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  document.title = `${to.meta.title || ''} · 链安智御`
  const store = useUserStore()
  if (store.token && typeof store.restoreSession === 'function') await store.restoreSession()
  if (to.matched.some(record => record.meta.requiresAuth) && !store.isLogin) {
    return { path: '/login', query: { redirect: to.path } }
  }
  if (to.path === '/login' && store.isLogin) return getDefaultRoute(store.userInfo)
  if (to.meta.permission && !hasPermission(store.userInfo, to.meta.permission)) {
    return { path: '/access-denied', query: { from: to.path }, replace: true }
  }
})

export default router
