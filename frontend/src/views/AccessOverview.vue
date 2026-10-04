<template>
  <div class="page-container">
    <PageHeading title="我的权限" description="查看当前账号的操作权限、数据密级与业务授权边界。" />
    <AccessScope />
    <section class="card">
      <div class="panel-heading"><div><h2 class="panel-title">当前账号</h2><p class="panel-description">访问数据需要同时满足角色权限、数据密级和业务范围。</p></div><el-tag type="success" size="small">已认证</el-tag></div>
      <dl class="account-facts"><div><dt>账号</dt><dd class="data-number">{{ store.username }}</dd></div><div><dt>业务角色</dt><dd>{{ store.role }}</dd></div><div><dt>数据密级上限</dt><dd>{{ store.dataLevel }}</dd></div><div><dt>授权组织</dt><dd>{{ ['管理员', '系统管理员'].includes(store.role) ? '平台运维' : store.userInfo?.department || '零售业务部' }}</dd></div></dl>
      <div class="permission-list"><div v-for="item in capabilities" :key="item.key"><span>{{ item.label }}</span><el-tag :type="hasPermission(store.userInfo, item.key) ? 'success' : 'info'" size="small">{{ hasPermission(store.userInfo, item.key) ? '已开放' : '未开放' }}</el-tag></div></div>
    </section>
    <section class="card section-gap">
      <div class="panel-heading"><div><h2 class="panel-title">角色与数据密级分别控制什么</h2><p class="panel-description">高密级不会自动增加操作权限，系统管理权限也不会自动授予客户数据。</p></div></div>
      <div class="table-scroll"><el-table :data="roleRows"><el-table-column prop="role" label="角色" min-width="130" /><el-table-column prop="actions" label="开放功能" min-width="230" /><el-table-column prop="scope" label="数据边界" min-width="280" /></el-table></div>
      <div class="levels"><div v-for="level in levels" :key="level.id" :class="{ current: store.dataLevel === level.id }"><strong>{{ level.id }}</strong><span>{{ level.name }}</span><small>{{ level.description }}</small></div></div>
      <p class="level-footnote">账号可访问的具体记录以业务授权为准；未授权组织、其他账号的私有记录及超出密级的数据均不可访问。</p>
    </section>
  </div>
</template>
<script setup>
import PageHeading from '@/components/PageHeading.vue'
import AccessScope from '@/components/AccessScope.vue'
import { useUserStore } from '@/store/user'
import { hasPermission } from '@/config/permissions'
const store = useUserStore()
const capabilities = [{ key: 'business.read', label: '业务记录' }, { key: 'business.submit', label: '提交业务' }, { key: 'dashboard.read', label: '安全态势' }, { key: 'alerts.read', label: '授权告警' }, { key: 'risk.review', label: '风险复核' }, { key: 'audit.read', label: '审计对账' }, { key: 'simulation.run', label: '模拟攻防' }, { key: 'admin.manage', label: '系统管理' }]
const roleRows = [{ role: '柜员／客服', actions: '业务办理、处理结果、风险提示', scope: '本人发起且已授权的业务' }, { role: '风控审核员', actions: '安全态势、告警详情、风险复核', scope: '授权部门及密级以内的风险记录' }, { role: '审计人员', actions: '审计记录、证据、链路对账（只读）', scope: '授权审计范围及密级以内的记录' }, { role: '系统管理员', actions: '账号、角色、策略、运行配置', scope: '平台配置；客户数据需要独立业务授权' }]
const levels = [{ id: 'L1', name: '公开数据', description: '公开产品与服务信息' }, { id: 'L2', name: '内部数据', description: '内部业务记录与风险摘要' }, { id: 'L3', name: '敏感数据', description: '授权交易与审计证据' }, { id: 'L4', name: '高度敏感', description: '核心敏感资料与安全配置' }]
</script>
<style scoped>
.account-facts { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 20px; padding-bottom: 22px; border-bottom: 1px solid var(--border); margin: 0 0 20px; }.account-facts dt { color: var(--text-sub); font-size: 12px; }.account-facts dd { margin: 6px 0 0; font-size: 15px; font-weight: 600; overflow-wrap: anywhere; }.permission-list { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px 28px; }.permission-list > div { display: flex; align-items: center; justify-content: space-between; gap: 10px; font-size: 13px; padding: 8px 0; }.levels { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); margin-top: 24px; border-top: 1px solid var(--border); }.levels > div { display: grid; gap: 6px; padding: 20px 16px; }.levels > div + div { border-left: 1px solid var(--border); }.levels strong { font-family: var(--font-data); font-size: 20px; }.levels small, .level-footnote { color: var(--text-sub); font-size: 12px; }.levels .current { background: var(--brand-light); color: var(--brand); }.level-footnote { margin: 16px 0 0; }@media(max-width: 850px){.permission-list,.account-facts,.levels { grid-template-columns: repeat(2, minmax(0, 1fr)); }}@media(max-width:480px){.permission-list { grid-template-columns: minmax(0, 1fr); }.levels > div { padding:16px 10px; }}
</style>
