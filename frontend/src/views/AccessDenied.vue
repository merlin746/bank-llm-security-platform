<template>
  <div class="page-container">
    <section class="card empty-state denied">
      <el-icon><Lock /></el-icon><h1 class="page-title">此页面不在您的授权范围内</h1>
      <p class="empty-description">当前身份为 {{ store.role }}，数据密级为 {{ store.dataLevel }}。如需访问，请联系管理员按职责申请授权。</p>
      <div class="page-actions"><el-button type="primary" @click="router.replace(getDefaultRoute(store.userInfo))">返回我的工作台</el-button><el-button @click="router.push('/my-access')">查看我的权限</el-button></div>
    </section>
  </div>
</template>
<script setup>
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/user'
import { getDefaultRoute } from '@/config/permissions'
const router = useRouter()
const store = useUserStore()
</script>
<style scoped>.denied { min-height: 420px; }.denied > .el-icon { color: var(--brand); font-size: 42px; }.denied h1 { font-size: 24px; }.denied p { max-width: 56ch; }</style>
