<template>
  <el-container class="layout">
    <!-- 侧边栏 -->
    <el-aside width="220px" class="aside">
      <div class="logo">
        <el-icon :size="22" color="#1e6fff"><Lock /></el-icon>
        <span>链安智御</span>
      </div>
      <el-menu
        :default-active="activeMenu"
        router
        background-color="#001529"
        text-color="#bfcbd9"
        active-text-color="#ffffff"
      >
        <el-menu-item index="/dashboard">
          <el-icon><DataAnalysis /></el-icon>
          <span>安全态势大屏</span>
        </el-menu-item>
        <el-menu-item index="/attack-defense">
          <el-icon><Aim /></el-icon>
          <span>模拟攻防测试</span>
        </el-menu-item>
        <el-menu-item index="/audit-topology">
          <el-icon><Connection /></el-icon>
          <span>审计溯源对账</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <!-- 顶栏 -->
      <el-header class="header">
        <div class="header-title">{{ $route.meta.title }}</div>
        <div class="header-user">
          <el-tag size="small" effect="plain">{{ userStore.role }}</el-tag>
          <span class="username">{{ userStore.username }}</span>
          <el-button link type="primary" @click="handleLogout">退出</el-button>
        </div>
      </el-header>

      <!-- 内容区 -->
      <el-main class="main">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/store/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const activeMenu = computed(() => route.path)

function handleLogout() {
  userStore.logout()
  ElMessage.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  background: #001529;
}
.logo {
  height: 60px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 20px;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}
.aside :deep(.el-menu) {
  border-right: none;
}
.header {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}
.header-title {
  font-size: 16px;
  font-weight: 600;
}
.header-user {
  display: flex;
  align-items: center;
  gap: 10px;
}
.username {
  color: #303133;
}
.main {
  background: #f5f7fa;
}
</style>
