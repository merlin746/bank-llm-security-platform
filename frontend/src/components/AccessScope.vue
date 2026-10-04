<template>
  <section class="access-scope" aria-label="当前账号授权范围">
    <div class="scope-identity"><el-icon aria-hidden="true"><Lock /></el-icon><strong>{{ store.role }}</strong><el-tag size="small" effect="plain">{{ store.dataLevel }} · {{ levelNames[store.dataLevel] || '未授权' }}</el-tag></div>
    <p>{{ scopeText }}</p>
    <router-link to="/my-access">查看我的权限<el-icon aria-hidden="true"><ArrowRight /></el-icon></router-link>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useUserStore } from '@/store/user'
const store = useUserStore()
const levelNames = { L1: '公开', L2: '内部', L3: '敏感', L4: '高度敏感' }
const scopeText = computed(() => {
  const user = store.userInfo || {}
  if (user.role === '系统管理员') return '账号、角色、策略与运行配置；客户业务数据需另行授权。'
  if (user.role === '柜员／客服') return '仅本人发起的授权业务；处理结果与风险提示。'
  if (user.role === '审计人员') return `${(user.scope?.departments || []).join('、') || '未授权组织'}的审计记录、证据与对账结果；只读访问。`
  if (user.role === '风控审核员') return `${user.department || '零售业务部'}授权范围内、密级不高于 ${store.dataLevel} 的态势与告警。`
  return '仅本人获准访问的只读业务记录。'
})
</script>

<style scoped>
.access-scope { display: flex; flex-wrap: wrap; align-items: center; gap: 10px 22px; padding: 14px 18px; margin: -4px 0 24px; background: var(--surface); border: 1px solid var(--border); border-radius: 10px; }
.scope-identity { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; font-size: 12px; }
.scope-identity > .el-icon { color: var(--brand); font-size: 16px; }
.access-scope p { flex: 1; margin: 0; font-size: 12px; color: var(--text-sub); min-width: 190px; }
.access-scope > a { display: inline-flex; align-items: center; gap: 6px; min-height: 30px; font-size: 12px; text-decoration: none; }
</style>
