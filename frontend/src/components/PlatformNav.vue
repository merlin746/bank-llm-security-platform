<template>
  <nav class="platform-nav" aria-label="主要导航" :style="{ '--nav-active-index': activeIndex }">
    <span v-if="activeIndex >= 0" class="nav-selection" aria-hidden="true" />
    <router-link v-for="item in items" :key="item.path" :to="item.path" @click="$emit('navigate')">
      <el-icon class="nav-icon" aria-hidden="true"><component :is="item.icon" /></el-icon>
      <span class="nav-label">{{ item.label }}</span>
      <el-icon class="nav-arrow" aria-hidden="true"><ArrowRight /></el-icon>
    </router-link>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/store/user'
import { hasPermission } from '@/config/permissions'

defineEmits(['navigate'])
const route = useRoute()
const store = useUserStore()
const allItems = [
  { path: '/business', label: '业务工作台', icon: 'User', permission: 'business.read' },
  { path: '/dashboard', label: '授权安全态势', icon: 'DataAnalysis', permission: 'dashboard.read' },
  { path: '/risk-review', label: '告警与风险复核', icon: 'Warning', permission: 'risk.review' },
  { path: '/audit-topology', label: '审计溯源对账', icon: 'Connection', permission: 'audit.read' },
  { path: '/administration', label: '系统管理', icon: 'Setting', permission: 'admin.manage' },
  { path: '/attack-defense', label: '模拟攻防测试', icon: 'Aim', permission: 'simulation.run' },
  { path: '/my-access', label: '我的权限', icon: 'Lock' }
]
const items = computed(() => allItems.filter(item => !item.permission || hasPermission(store.userInfo, item.permission)))
const activeIndex = computed(() => items.value.findIndex(item => item.path === route.path))
</script>

<style scoped>
.platform-nav { position: relative; isolation: isolate; display: grid; gap: 8px; }
.nav-selection { position: absolute; inset: 0 0 auto; height: 48px; border-radius: 8px; background: #24436a; pointer-events: none; transform: translateY(calc(var(--nav-active-index) * 56px)); transition: transform 240ms cubic-bezier(0.16, 1, 0.3, 1); }
.platform-nav a { position: relative; z-index: 1; isolation: isolate; overflow: hidden; display: flex; align-items: center; gap: 12px; height: 48px; padding: 0 14px; border-radius: 8px; color: #aebfd5; text-decoration: none; font-size: 14px; transition: background-color 180ms, color 180ms, transform 120ms cubic-bezier(0.16, 1, 0.3, 1); }
.platform-nav a:focus-visible { background: #1d304d; color: #f0f5ff; }
.platform-nav a.router-link-active { color: #f0f5ff; font-weight: 600; }
.platform-nav a.router-link-active:focus-visible { background: transparent; }
.nav-icon, .nav-label, .nav-arrow { position: relative; z-index: 1; transition: transform 180ms cubic-bezier(0.16, 1, 0.3, 1), opacity 180ms; }
.nav-label { white-space: nowrap; }
.platform-nav .el-icon { font-size: 19px; }
.platform-nav .nav-arrow { margin-left: auto; opacity: 0; font-size: 13px; transform: translateX(-4px); }
.platform-nav .router-link-active .nav-arrow { opacity: 1; transform: translateX(0); }
@media (hover: hover) {
  .platform-nav a:hover { background: #1d304d; color: #f0f5ff; }
  .platform-nav a.router-link-active:hover { background: #2c507c; }
  .platform-nav a:hover .nav-arrow { opacity: 1; transform: translateX(0); }
}
@media (hover: hover) and (prefers-reduced-motion: no-preference) {
  .platform-nav a:hover .nav-icon, .platform-nav a:hover .nav-label { transform: translateX(3px); }
}
@media (prefers-reduced-motion: no-preference) {
  .platform-nav a:active { transform: scale(0.98); }
}
@media (prefers-reduced-motion: reduce) {
  .nav-selection, .platform-nav a, .nav-icon, .nav-label, .nav-arrow { transition: none; }
  .platform-nav .nav-arrow { transform: none; }
}
</style>
