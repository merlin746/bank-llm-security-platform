<template>
  <div class="page-container">
    <PageHeading :title="canSubmit ? '我的业务工作台' : '授权业务记录'" :description="canSubmit ? '处理本人授权业务，查看处理结果与风险提示。' : '查看授权组织与密级范围内的业务处理记录。'">
      <el-button :loading="pending" @click="refresh"><el-icon v-if="!pending"><Refresh /></el-icon>刷新记录</el-button>
    </PageHeading>
    <AccessScope />
    <div v-if="canSubmit" class="business-grid">
      <section class="card">
        <div class="panel-heading"><div><h2 class="panel-title">发起业务请求</h2><p class="panel-description">提交后查看安全处理结果。</p></div></div>
        <el-alert v-if="submitError" :title="submitError" type="error" show-icon :closable="false" class="resource-error" />
        <el-form ref="formRef" :model="form" :rules="rules" label-position="top" :disabled="submitting" @submit.prevent="submit">
          <el-form-item label="业务类型" prop="businessType"><el-select v-model="form.businessType" aria-label="业务类型"><el-option label="产品与服务咨询" value="产品与服务咨询" /><el-option label="业务办理指引" value="业务办理指引" /><el-option label="客户服务回复" value="客户服务回复" /></el-select></el-form-item>
          <el-form-item label="业务数据密级" prop="dataLevel"><el-select v-model="form.dataLevel" aria-label="业务数据密级"><el-option v-for="level in allowedLevels" :key="level" :label="level + ' · ' + levelNames[level]" :value="level" /></el-select></el-form-item>
          <el-form-item label="业务内容" prop="prompt"><el-input v-model="form.prompt" name="businessPrompt" type="textarea" :rows="5" maxlength="2000" show-word-limit placeholder="例如：请说明本行公开的储蓄产品办理流程" /></el-form-item>
          <el-button type="primary" native-type="submit" :loading="submitting" class="business-submit">{{ submitting ? '正在处理业务' : '提交业务请求' }}<el-icon v-if="!submitting" class="action-arrow"><ArrowRight /></el-icon></el-button>
        </el-form>
      </section>
      <section class="card business-result" :aria-busy="submitting">
        <div class="panel-heading"><h2 class="panel-title">本次处理结果</h2><el-tag v-if="latest" :type="tone(latest)" size="small">{{ statusLabel(latest) }}</el-tag></div>
        <el-skeleton v-if="submitting" :rows="7" animated />
        <template v-else-if="latest">
          <DetectionNotice v-if="latest.detection" :detection="latest.detection" />
          <dl class="business-facts"><div><dt>请求编号</dt><dd class="data-number">{{ latest.requestId }}</dd></div><div><dt>数据密级</dt><dd>{{ latest.dataLevel || form.dataLevel }}</dd></div></dl>
          <h3>处理结果</h3><p class="business-copy">{{ latest.result || (latest.verdict === 'block' ? '请求已被安全策略拦截，请核实业务内容后重新提交。' : '业务请求已完成安全检查，已生成授权范围内的业务办理指引。') }}</p>
          <div class="business-risk" :class="{ blocked: latest.verdict === 'block' }"><el-icon><Warning v-if="latest.verdict === 'block'" /><CircleCheck v-else /></el-icon><p>{{ riskHint(latest) }}</p></div>
          <p class="panel-description">仅展示本次业务结果。风险复核由授权风控人员处理。</p>
        </template>
        <div v-else class="empty-state"><el-icon><DataLine /></el-icon><h3>提交后查看处理结果</h3><p class="empty-description">处理结果包含业务反馈与风险提示，便于您继续完成客户服务。</p></div>
      </section>
    </div>
    <section class="card" :class="{ 'section-gap': canSubmit }">
      <div class="panel-heading"><div><h2 class="panel-title">{{ canSubmit ? '我的业务记录' : '授权业务记录' }}</h2><p class="panel-description">{{ canSubmit ? '仅显示本人发起的业务' : '仅显示符合当前组织授权与数据密级的记录' }} · {{ filteredRecords.length }} 条</p></div><el-input v-model="search" class="record-search" aria-label="搜索业务记录" placeholder="搜索请求编号或业务内容" clearable><template #prefix><el-icon><Search /></el-icon></template></el-input></div>
      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="resource-error" />
      <el-skeleton v-if="loading" :rows="5" animated />
      <div v-else class="table-scroll" role="region" aria-label="业务记录表" tabindex="0"><el-table :data="filteredRecords" row-key="requestId" empty-text="暂无授权业务记录">
        <el-table-column label="请求编号" min-width="210"><template #default="{ row }"><span class="data-number record-id">{{ row.requestId }}</span></template></el-table-column>
        <el-table-column prop="time" label="提交时间" min-width="175" /><el-table-column label="业务内容" min-width="230"><template #default="{ row }">{{ row.businessType || row.action || row.prompt || '业务处理' }}</template></el-table-column>
        <el-table-column prop="dataLevel" label="密级" width="80" /><el-table-column v-if="!canSubmit" prop="owner" label="提交账号" min-width="120" /><el-table-column label="状态" width="110"><template #default="{ row }"><el-tag :type="tone(row)" size="small">{{ statusLabel(row) }}</el-tag></template></el-table-column>
        <el-table-column label="操作" width="105" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="selected = row">查看结果</el-button></template></el-table-column>
      </el-table></div>
    </section>
    <section v-if="selected" class="card section-gap" aria-label="业务处理详情"><div class="panel-heading"><h2 class="panel-title">业务处理详情</h2><el-button @click="selected = null">收起详情</el-button></div><dl class="business-facts"><div><dt>请求编号</dt><dd class="data-number">{{ selected.requestId }}</dd></div><div><dt>状态 / 密级</dt><dd>{{ statusLabel(selected) }} / {{ selected.dataLevel }}</dd></div></dl><p class="business-copy">{{ selected.result || selected.prompt || selected.action || '已完成授权业务的安全处理。' }}</p><div class="business-risk" :class="{ blocked: selected.verdict === 'block' }"><el-icon><Warning /></el-icon><p>{{ riskHint(selected) }}</p></div></section>
  </div>
</template>
<script setup>
import { computed, reactive, ref } from 'vue'
import { getBusinessRequests, submitBusinessRequest } from '@/api/business'
import { useResource } from '@/composables/useResource'
import { useUserStore } from '@/store/user'
import { hasPermission } from '@/config/permissions'
import PageHeading from '@/components/PageHeading.vue'
import AccessScope from '@/components/AccessScope.vue'
import DetectionNotice from '@/components/DetectionNotice.vue'
const store = useUserStore()
const canSubmit = computed(() => hasPermission(store.userInfo, 'business.submit'))
const levelNames = { L1: '公开数据', L2: '内部数据', L3: '敏感数据', L4: '高度敏感' }
const allowedLevels = computed(() => Object.keys(levelNames).filter(level => Number(level.slice(1)) <= Number(store.dataLevel.slice(1))))
const form = reactive({ businessType: '产品与服务咨询', dataLevel: 'L1', prompt: '' })
const rules = { prompt: [{ required: true, whitespace: true, message: '请输入需要处理的业务内容', trigger: 'blur' }] }
const formRef = ref()
const submitting = ref(false)
const submitError = ref('')
const latest = ref(null)
const selected = ref(null)
const search = ref('')
const { data, pending, loading, error, refresh } = useResource(async () => { const response = await getBusinessRequests(); if (!Array.isArray(response.data)) throw new Error('业务记录暂时无法读取，请重新加载'); return response.data })
const filteredRecords = computed(() => (data.value || []).filter(row => !search.value.trim() || [row.requestId, row.prompt, row.businessType, row.action].some(value => String(value || '').toLowerCase().includes(search.value.trim().toLowerCase()))))
function tone(row) { return row.verdict === 'block' || row.status === 'blocked' ? 'warning' : 'success' }
function statusLabel(row) { return row.verdict === 'block' || row.status === 'blocked' ? '已拦截' : '已完成' }
function riskHint(row) { return row.riskTip || (row.verdict === 'block' ? '存在风险，已在处理前阻断。请按业务流程核实后重新提交。' : '本次请求未发现风险，输出仅限当前授权的数据范围。') }
async function submit() {
  if (submitting.value || !canSubmit.value || !await formRef.value.validate().catch(() => false)) return
  submitting.value = true; submitError.value = ''; latest.value = null
  try { const response = await submitBusinessRequest({ ...form, prompt: form.prompt.trim(), action: form.businessType }); latest.value = response.data; await refresh() }
  catch (reason) { submitError.value = reason.message || '业务处理失败，请重试' }
  finally { submitting.value = false }
}
</script>
<style scoped>
.business-grid { display:grid; grid-template-columns:minmax(0,.95fr) minmax(0,1.05fr); gap:22px; align-items:start; }.el-select { width:100%; }.business-submit { width:100%; gap:8px; }.business-result { min-height:445px; }.business-result .empty-state { min-height:330px; }.business-facts { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:20px; margin:0 0 24px; }.business-facts dt { font-size:12px; color:var(--text-sub); }.business-facts dd { margin:6px 0 0; font-size:13px; overflow-wrap:anywhere; }.business-result h3 { margin:24px 0 10px; font-size:14px; }.business-copy { font-size:13px; white-space:pre-wrap; overflow-wrap:anywhere; line-height:1.8; }.business-risk { display:flex; align-items:flex-start; gap:10px; padding:16px; margin:24px 0 16px; border-radius:10px; background:var(--success-light); color:var(--success); }.business-risk.blocked { background:var(--warning-light); color:var(--warning); }.business-risk .el-icon { margin-top:3px; flex-shrink:0; }.business-risk p { margin:0; font-size:12px; }.record-search { width:260px; }.record-id { font-size:11px; }@media(max-width:1000px){.business-grid { grid-template-columns:minmax(0,1fr); }.business-result { min-height:270px; }.business-result .empty-state { min-height:210px; }}@media(max-width:640px){.record-search { width:100%; }.business-facts { grid-template-columns:minmax(0,1fr); }}
</style>
