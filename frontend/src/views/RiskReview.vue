<template>
  <div class="page-container">
    <PageHeading title="告警与风险复核" description="查看授权业务范围内的风险证据，记录复核结论。"><el-button :loading="pending" @click="refresh"><el-icon v-if="!pending"><Refresh /></el-icon>刷新告警</el-button></PageHeading>
    <AccessScope />
    <section class="card">
      <div class="panel-heading"><div><h2 class="panel-title">授权告警</h2><p class="panel-description">{{ filteredAlerts.length }} 条记录 · 仅统计授权范围</p></div><div class="review-filters"><el-input v-model="search" placeholder="搜索告警内容或请求编号" clearable aria-label="搜索授权告警"><template #prefix><el-icon><Search /></el-icon></template></el-input><el-select v-model="statusFilter" aria-label="筛选复核状态"><el-option label="全部状态" value="all" /><el-option label="待复核" value="pending" /><el-option label="已复核" value="reviewed" /></el-select></div></div>
      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="resource-error" />
      <el-skeleton v-if="loading" :rows="7" animated />
      <div v-else class="table-scroll" role="region" aria-label="授权告警表" tabindex="0"><el-table :data="filteredAlerts" row-key="id" empty-text="暂无符合条件的授权告警">
        <el-table-column label="编号" width="85"><template #default="{ row }"><span class="data-number">{{ row.id }}</span></template></el-table-column><el-table-column prop="time" label="发生时间" min-width="180" /><el-table-column label="风险类型" min-width="125"><template #default="{ row }"><el-tag :type="row.type === 'hash-mismatch' ? 'danger' : 'warning'" size="small">{{ typeLabel(row.type) }}</el-tag></template></el-table-column><el-table-column prop="message" label="告警内容" min-width="280" /><el-table-column prop="dataLevel" label="密级" width="80" /><el-table-column label="复核状态" width="110"><template #default="{ row }"><el-tag :type="isReviewed(row) ? 'success' : 'warning'" size="small">{{ isReviewed(row) ? '已复核' : '待复核' }}</el-tag></template></el-table-column><el-table-column label="操作" width="110" fixed="right"><template #default="{ row }"><el-button link type="primary" @click="selectAlert(row.id)">查看与复核</el-button></template></el-table-column>
      </el-table></div>
    </section>
    <section v-if="selectedId || selectedRequestId" ref="detailPanel" class="card section-gap review-detail" aria-label="风险复核详情" tabindex="-1">
      <div class="panel-heading"><h2 class="panel-title">{{ selectedId ? '告警详情与风险复核' : '请求详情' }}</h2><el-button @click="closeDetail">收起详情</el-button></div>
      <el-skeleton v-if="detailPending" :rows="7" animated />
      <div v-else-if="detailError"><el-alert :title="detailError" type="error" show-icon :closable="false" /><el-button class="section-gap" @click="loadDetail">重新加载详情</el-button></div>
      <template v-else-if="!detail && requestDetail">
        <RequestDetail :request="requestDetail" />
        <div v-if="requestDetail.alertIds?.length" class="section-gap"><span>关联告警 </span><el-button v-for="id in requestDetail.alertIds" :key="id" link type="primary" @click="selectAlert(id)">查看告警 {{ id }}</el-button></div>
      </template>
      <template v-else-if="detail">
        <dl class="review-facts"><div><dt>告警编号</dt><dd class="data-number">{{ detail.id }}</dd></div><div><dt>风险类型</dt><dd>{{ typeLabel(detail.type) }}</dd></div><div><dt>业务密级</dt><dd>{{ detail.dataLevel }}</dd></div><div><dt>授权组织</dt><dd>{{ detail.department || store.userInfo?.department }}</dd></div></dl>
        <h3>告警内容</h3><p class="review-copy">{{ detail.message }}</p><h3>风险证据</h3><p class="review-evidence">{{ detail.evidence || '此告警未保留证据' }}</p><h3>处置建议</h3><p class="review-copy">{{ detail.recommendation || '核对业务授权与请求来源后记录复核结论。' }}</p>
        <RequestDetail v-if="requestDetail" class="linked-request" :request="requestDetail" />
        <el-alert v-if="requestError" :title="requestError" type="warning" :closable="false" class="section-gap" />
        <div class="review-form section-gap">
          <h3>记录复核结论</h3><p class="panel-description">复核用于记录风险判断，不会解除拦截或增加账号权限。</p>
          <el-alert v-if="reviewError" :title="reviewError" type="error" show-icon :closable="false" class="section-gap" />
          <el-alert v-if="reviewSuccess" title="复核结论已保存" type="success" show-icon :closable="false" class="section-gap" />
          <template v-if="detail.review || isReviewed(detail)"><p class="review-copy">结论：{{ (detail.review?.decision || detail.reviewDecision) === 'dismissed' ? '排除风险' : '确认风险' }}</p><p class="review-copy">{{ detail.review?.note || detail.review?.comment || detail.reviewNote || '复核记录已保存。' }}</p></template>
          <el-form v-else ref="reviewForm" :model="form" :rules="rules" label-position="top" :disabled="saving" @submit.prevent="saveReview"><el-form-item label="复核结论" prop="decision"><el-select v-model="form.decision" aria-label="复核结论"><el-option label="确认风险" value="confirmed" /><el-option label="排除风险" value="dismissed" /></el-select></el-form-item><el-form-item label="复核说明" prop="note"><el-input v-model="form.note" type="textarea" :rows="3" maxlength="1000" show-word-limit placeholder="说明核查依据和建议处置方式" /></el-form-item><el-button type="primary" native-type="submit" :loading="saving">保存复核结论</el-button></el-form>
        </div>
      </template>
    </section>
  </div>
</template>
<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getAlerts, getAlertDetail, getRequestDetail } from '@/api/audit'
import { submitRiskReview } from '@/api/risk'
import { useResource } from '@/composables/useResource'
import { useUserStore } from '@/store/user'
import PageHeading from '@/components/PageHeading.vue'
import AccessScope from '@/components/AccessScope.vue'
import RequestDetail from '@/components/RequestDetail.vue'
const store = useUserStore()
const route = useRoute()
const router = useRouter()
const search = ref('')
const statusFilter = ref('all')
const selectedId = computed(() => typeof route.query.alertId === 'string' ? route.query.alertId : '')
const selectedRequestId = computed(() => typeof route.query.requestId === 'string' ? route.query.requestId : '')
const detail = ref(null), requestDetail = ref(null), detailPending = ref(false), detailError = ref(''), requestError = ref(''), detailPanel = ref()
const saving = ref(false), reviewError = ref(''), reviewSuccess = ref(false), reviewForm = ref()
const form = reactive({ decision: 'confirmed', note: '' })
const rules = { note: [{ required: true, whitespace: true, message: '请填写复核依据', trigger: 'blur' }] }
let version = 0
const { data, pending, loading, error, refresh } = useResource(async () => { const response = await getAlerts(); if (!Array.isArray(response.data)) throw new Error('告警数据暂时无法读取'); return response.data })
function isReviewed(row) { return !!(row.review || row.reviewDecision || ['resolved', 'reviewed', 'confirmed', 'dismissed'].includes(row.status)) }
const filteredAlerts = computed(() => (data.value || []).filter(row => (statusFilter.value === 'all' || (statusFilter.value === 'reviewed' ? isReviewed(row) : !isReviewed(row))) && (!search.value.trim() || [row.message, row.requestId, row.id, typeLabel(row.type)].some(value => String(value || '').toLowerCase().includes(search.value.trim().toLowerCase())))))
function typeLabel(type) { return { 'hash-mismatch': 'Hash 不一致', privilege: '越权访问', jailbreak: '注入攻击' }[type] || type }
function selectAlert(id) { router.push({ query: { alertId: String(id) } }) }
function closeDetail() { router.replace({ query: {} }) }
async function loadDetail() {
  const current = ++version; const id = selectedId.value
  detail.value = null; requestDetail.value = null; detailError.value = ''; requestError.value = ''; reviewError.value = ''; reviewSuccess.value = false; form.note = ''; form.decision = 'confirmed'
  detailPending.value = false
  if (!id && !selectedRequestId.value) return
  detailPending.value = true
  try {
    if (!id) { const request = await getRequestDetail(selectedRequestId.value); if (current === version) requestDetail.value = request.data }
    else { const response = await getAlertDetail(id); if (current !== version) return; detail.value = response.data; if (detail.value?.requestId) { try { const request = await getRequestDetail(detail.value.requestId); if (current === version) requestDetail.value = request.data } catch (reason) { if (current === version) requestError.value = reason.message || '关联请求不在授权范围内' } } }
  }
  catch (reason) { if (current === version) detailError.value = reason.message || '告警详情加载失败' }
  finally { if (current === version) { detailPending.value = false; await nextTick(); detailPanel.value?.focus({ preventScroll: true }); detailPanel.value?.scrollIntoView({ block: 'start' }) } }
}
watch([selectedId, selectedRequestId], loadDetail, { immediate: true })
onBeforeUnmount(() => { version += 1 })
async function saveReview() {
  if (saving.value || !await reviewForm.value.validate().catch(() => false)) return
  const current = version; const id = detail.value.id
  saving.value = true; reviewError.value = ''
  try { await submitRiskReview({ alertId: id, decision: form.decision, note: form.note.trim() }); if (current !== version) return; await loadDetail(); if (selectedId.value === String(id) && detail.value?.review) reviewSuccess.value = true; await refresh() }
  catch (reason) { if (current === version) reviewError.value = reason.message || '复核保存失败，请重试' }
  finally { saving.value = false }
}
</script>
<style scoped>
.review-filters { display:flex; gap:10px; }.review-filters .el-input { width:245px; }.review-filters .el-select { width:125px; }.review-detail { scroll-margin-top:90px; }.review-facts { display:grid; grid-template-columns:repeat(4,minmax(0,1fr)); gap:20px; margin:0 0 24px; }.review-facts dt { color:var(--text-sub); font-size:12px; }.review-facts dd { margin:5px 0 0; font-size:13px; }.review-detail h3 { font-size:14px; margin:24px 0 10px; }.review-copy,.review-evidence { font-size:13px; white-space:pre-wrap; overflow-wrap:anywhere; }.review-evidence { padding:16px; border-radius:8px; background:var(--surface-sub); }.linked-request { border-top:1px solid var(--border); padding-top:24px; margin-top:24px; }.review-form { border-top:1px solid var(--border); padding-top:8px; }.review-form .el-form { max-width:720px; margin-top:22px; }.review-form .el-select { width:100%; }@media(max-width:850px){.review-facts { grid-template-columns:repeat(2,minmax(0,1fr)); }.review-filters { width:100%; }.review-filters .el-input { flex:1; width:auto; }}
</style>
