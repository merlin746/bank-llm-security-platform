<template>
  <div v-feedback class="layout">
    <a class="skip-link" href="#main-content">跳到页面内容</a>
    <aside class="aside">
      <BrandLogo inverse class="sidebar-brand" />
      <div class="sidebar-identity"><strong>{{ userStore.role }}</strong><span>{{ access.scopeLabel }}</span></div>
      <PlatformNav />
      <div class="aside-footer">
        <span>全链路安全管控</span>
        <p>事前拦截 · 事后溯源</p>
      </div>
    </aside>

    <div class="workspace">
      <header class="header">
        <div class="header-location">
          <el-button class="mobile-menu icon-button" aria-label="打开导航菜单" :aria-expanded="drawerOpen" @click="drawerOpen = true">
            <el-icon><Menu /></el-icon>
          </el-button>
          <span class="workspace-label">安全管控工作台</span>
          <span class="breadcrumb-separator" aria-hidden="true">/</span>
          <span class="header-title">{{ route.meta.title }}</span>
        </div>
        <div class="header-user">
          <span class="environment" :class="{ 'is-demo': isDemo }">{{ isDemo ? '演示数据' : '后端联调' }}</span>
          <el-button class="icon-button" :aria-label="isDark ? '切换浅色主题' : '切换深色主题'" :title="isDark ? '切换浅色主题' : '切换深色主题'" @click="toggleTheme">
            <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
          </el-button>
          <div class="user-avatar" aria-hidden="true">{{ userStore.username.slice(0, 1).toUpperCase() }}</div>
          <div class="user-info"><span>{{ userStore.username }}</span><small>{{ userStore.role }} / {{ userStore.dataLevel }}</small></div>
          <el-button class="icon-button" aria-label="退出登录" title="退出登录" @click="handleLogout"><el-icon><SwitchButton /></el-icon></el-button>
        </div>
      </header>
      <main id="main-content" class="main" tabindex="-1"><router-view :key="userStore.username + ':' + route.path" /></main>
    </div>

    <el-drawer v-model="drawerOpen" title="工作台导航" direction="ltr" size="280px" class="navigation-drawer">
      <div v-feedback>
        <BrandLogo inverse compact class="drawer-brand" />
        <PlatformNav @navigate="drawerOpen = false" />
      </div>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/store/user'
import { useTheme } from '@/composables/useTheme'
import { useMock } from '@/mock'
import PlatformNav from '@/components/PlatformNav.vue'
import BrandLogo from '@/components/BrandLogo.vue'
import { getRolePresentation } from '@/config/permissions'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { isDark, toggleTheme } = useTheme()
const isDemo = useMock()
const drawerOpen = ref(false)
const access = computed(() => getRolePresentation(userStore.userInfo))

async function handleLogout() {
  await userStore.logout()
  ElMessage.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.layout { display: grid; grid-template-columns: 236px minmax(0, 1fr); min-height: 100dvh; }
.aside { position: sticky; top: 0; height: 100dvh; display: flex; flex-direction: column; padding: 30px 18px 24px; background: #12233c; z-index: var(--layer-sidebar); }
.sidebar-brand { margin: 0 8px 28px; }
.sidebar-identity { display: flex; flex-direction: column; gap: 5px; margin: 0 12px 26px; padding-bottom: 20px; border-bottom: 1px solid #314158; color: #f0f5ff; font-size: 13px; }
.sidebar-identity span { color: #aebfd5; font-size: 11px; }
.drawer-brand { margin: 0 12px 28px; }
.aside-footer { margin-top: auto; border-top: 1px solid #314158; padding: 22px 12px 0; font-size: 12px; color: #aebfd5; }
.aside-footer p { margin: 4px 0 0; font-size: 11px; }
.workspace, .main { min-width: 0; }
.header { position: sticky; top: 0; z-index: var(--layer-header); display: flex; align-items: center; justify-content: space-between; gap: 16px; min-height: 76px; padding: 12px 36px; background: var(--surface); border-bottom: 1px solid var(--border); }
.header-location, .header-user { display: flex; align-items: center; gap: 14px; min-width: 0; }
.workspace-label { color: var(--text-sub); font-size: 12px; white-space: nowrap; }
.breadcrumb-separator { color: var(--text-sub); }
.header-title { font-size: 13px; white-space: nowrap; }
.header-user { gap: 12px; flex-shrink: 0; }
.environment { color: var(--text-sub); font-size: 11px; padding: 3px 8px; border-radius: 5px; background: var(--surface-sub); white-space: nowrap; }
.environment.is-demo { color: var(--warning); background: var(--warning-light); }
.icon-button { width: 38px; padding: 0; margin-left: 0; color: var(--text-sub); background: transparent; border-color: transparent; font-size: 18px; }
.user-avatar { width: 34px; height: 34px; border-radius: 50%; display: grid; place-items: center; background: var(--brand-light); color: var(--brand); font-size: 14px; font-weight: 600; }
.user-info { display: grid; font-size: 12px; line-height: 1.6; }
.user-info small { color: var(--text-sub); font-size: 10px; }
.mobile-menu { display: none; }
.skip-link { position: fixed; left: 16px; top: 8px; z-index: 2100; background: var(--surface); padding: 8px 16px; border-radius: 6px; transform: translateY(-160%); }
.skip-link:focus { transform: translateY(0); }
@media (max-width: 1200px) { .workspace-label, .breadcrumb-separator { display: none; } .header { padding: 12px 24px; } }
@media (max-width: 900px) { .layout { grid-template-columns: minmax(0, 1fr); } .aside { display: none; } .mobile-menu { display: inline-flex; } .header { min-height: 68px; } }
@media (max-width: 640px) { .header { padding: 10px 12px; gap: 8px; } .header-location, .header-user { gap: 4px; } .header-title { font-size: 12px; } .user-info, .user-avatar { display: none; } .environment { font-size: 10px; padding: 3px 5px; } }
</style>

<style>
.navigation-drawer { background: #12233c; --el-drawer-bg-color: #12233c; --el-text-color-primary: #f0f5ff; --el-text-color-regular: #f0f5ff; }
.navigation-drawer .el-drawer__header { color: #f0f5ff; padding: 24px 24px 0; }
</style>
