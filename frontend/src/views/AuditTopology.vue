<template>
  <div class="page-container">
    <PageHeading title="审计溯源对账" description="只读查看授权审计记录、风险证据与节点指纹对账结果。">
      <span v-if="updatedAt" class="updated-time">更新于 {{ updateLabel }}</span>
      <el-button :loading="pending" @click="refresh"><el-icon v-if="!pending"><Refresh /></el-icon><span>刷新对账</span></el-button>
    </PageHeading>
    <AccessScope />
    <el-alert v-if="error" class="resource-error" :title="hasData ? '刷新失败，当前展示上次对账结果' : '审计数据加载失败'"
      :description="error + '。请检查服务连接后重新加载。'" type="error" show-icon :closable="false" />
    <section v-if="loading" class="card" aria-label="正在加载审计数据" aria-busy="true"><el-skeleton :rows="12" animated /></section>
    <template v-else-if="hasData">
      <div class="chain-status" :class="'is-' + chainState.tone" role="status">
        <el-icon><CircleCheck v-if="chainState.tone === 'success'" /><Warning v-else /></el-icon>
        <div><h2>{{ chainState.title }}</h2><p>{{ topology?.alert || chainState.description }}</p></div>
        <span class="node-count">{{ nodes.length }} 个节点</span>
      </div>
      <div class="audit-grid section-gap">
        <section class="card">
          <div class="panel-heading"><div><h2 class="panel-title">四节点 Hash 连贯性拓扑</h2><p class="panel-description">异常节点以菱形标记，可拖动与缩放查看</p></div></div>
          <HashTopology v-if="nodes.length" :topology="topology" />
          <el-empty v-else description="暂无节点记录，无法进行链路对账" :image-size="80" />
          <div class="topology-legend"><span><el-icon class="normal"><CircleCheck /></el-icon>正常</span><span><el-icon class="tampered"><CircleClose /></el-icon>篡改</span><span><el-icon><Remove /></el-icon>未知</span></div>
        </section>
        <section class="card">
          <div class="panel-heading"><h2 class="panel-title">节点 Hash 明细</h2></div>
          <div v-for="node in nodes" :key="node.id" class="hash-row" :class="{ tampered: node.status === 'tampered' }">
            <div class="hash-name"><span>{{ node.name }}</span><el-tag :type="statusType(node.status)" size="small">{{ statusLabel(node.status) }}</el-tag></div>
            <div class="hash-line"><code class="hash-value">{{ node.hash || '未返回 Hash' }}</code><el-button text class="copy-button" :disabled="!node.hash" :aria-label="'复制' + node.name + '的 Hash'" @click="copyHash(node.hash)"><el-icon><CopyDocument /></el-icon></el-button></div>
          </div>
          <el-empty v-if="!nodes.length" description="暂无节点指纹" :image-size="60" />
        </section>
      </div>
      <section class="card section-gap">
        <div class="panel-heading"><div><h2 class="panel-title">授权审计记录</h2><p class="panel-description">{{ data.requests.length }} 条记录 · 只读查证</p></div></div>
        <div class="table-scroll" role="region" aria-label="授权审计记录表" tabindex="0">
          <el-table :data="data.requests" row-key="requestId" empty-text="暂无授权审计记录">
            <el-table-column prop="time" label="发生时间" min-width="185" />
            <el-table-column prop="owner" label="业务账号" min-width="100" />
            <el-table-column prop="department" label="授权组织" min-width="130" />
            <el-table-column prop="dataLevel" label="密级" width="80" />
            <el-table-column label="处理结果" width="110"><template #default="{ row }">{{ row.verdict === 'block' ? '已拦截' : row.verdict === 'pass' ? '已放行' : '待核实' }}</template></el-table-column>
            <el-table-column label="请求编号与证据" min-width="230"><template #default="{ row }"><el-button link type="primary" class="data-number" @click="selectRequest(row.requestId)">{{ row.requestId }}</el-button></template></el-table-column>
          </el-table>
        </div>
      </section>
      <section class="card section-gap">
        <div class="panel-heading">
          <div><h2 class="panel-title">告警日志</h2><p class="panel-description">显示 {{ filteredAlerts.length }} 条记录</p></div>
          <el-input v-model="search" class="alert-search" aria-label="搜索告警日志" placeholder="搜索节点、类型、内容或请求编号" clearable><template #prefix><el-icon><Search /></el-icon></template></el-input>
        </div>
        <div class="table-scroll" role="region" aria-label="告警日志表，可横向滚动" tabindex="0">
          <el-table :data="filteredAlerts" row-key="id" empty-text="暂无符合条件的告警记录">
            <el-table-column prop="time" label="时间" min-width="195"><template #default="{ row }"><time class="alert-time data-number">{{ row.time }}</time></template></el-table-column>
            <el-table-column prop="node" label="节点" min-width="150" />
            <el-table-column label="类型" width="150"><template #default="{ row }"><el-tag size="small" :type="typeTag(row.type)">{{ typeLabel(row.type) }}</el-tag></template></el-table-column>
            <el-table-column prop="message" label="内容" min-width="300" />
            <el-table-column label="关联请求" min-width="210"><template #default="{ row }"><el-button v-if="row.requestId" link type="primary" class="request-id-button data-number" @click="selectRequest(row.requestId)">{{ row.requestId }}</el-button><span v-else class="muted">未关联</span></template></el-table-column>
            <el-table-column label="操作" width="105" fixed="right"><template #default="{ row }"><el-button link type="primary" :aria-label="'查看告警 ' + row.id + ' 详情'" @click="selectAlert(row.id)">查看详情</el-button></template></el-table-column>
          </el-table>
        </div>
      </section>
    </template>
    <div v-else class="card empty-state">
      <el-icon><Warning /></el-icon><h3>暂时无法获取审计数据</h3><p class="empty-description">检查服务连接后，重新加载节点指纹与告警日志。</p><el-button type="primary" :loading="pending" @click="refresh">重新加载</el-button>
    </div>
    <section v-if="selectionActive" ref="detailPanel" class="card section-gap detail-panel" aria-label="告警与请求详情" tabindex="-1">
      <div class="panel-heading"><h2 class="panel-title">{{ selectedAlertId ? '告警详情' : '请求详情' }}</h2><el-button @click="closeDetails">收起详情</el-button></div>
      <div v-if="alertPending" role="status" aria-label="正在加载告警详情"><el-skeleton :rows="5" animated /></div>
      <div v-else-if="alertError" class="detail-error" role="alert"><p>{{ alertError }}</p><el-button @click="loadSelection">重新加载告警</el-button></div>
      <template v-else-if="alertDetail">
        <dl class="alert-facts">
          <div><dt>告警编号</dt><dd class="data-number">{{ alertDetail.id }}</dd></div>
          <div><dt>发生时间</dt><dd>{{ alertDetail.time }}</dd></div>
          <div><dt>节点</dt><dd>{{ alertDetail.node }}</dd></div>
          <div><dt>类型</dt><dd><el-tag size="small" :type="typeTag(alertDetail.type)">{{ typeLabel(alertDetail.type) }}</el-tag></dd></div>
          <div><dt>严重程度</dt><dd>{{ severityLabel(alertDetail.severity) }}</dd></div>
          <div><dt>处理状态</dt><dd>{{ alertStatusLabel(alertDetail.status) }}</dd></div>
        </dl>
        <h3 class="detail-title">告警内容</h3><p class="detail-copy">{{ alertDetail.message }}</p>
        <h3 class="detail-title">检测依据</h3><p class="detail-evidence">{{ alertDetail.evidence || '此告警未保留检测依据' }}</p>
        <h3 class="detail-title">处置建议</h3><p class="detail-copy">{{ alertDetail.recommendation || '请核对节点记录并复核关联请求。' }}</p>
        <template v-if="alertDetail.review"><h3 class="detail-title">风险复核记录</h3><p class="detail-copy">{{ alertDetail.review.decision === 'dismissed' ? '排除风险' : '确认风险' }} · {{ alertDetail.review.reviewer }} · {{ alertDetail.review.time }}</p><p class="detail-copy">{{ alertDetail.review.note }}</p></template>
      </template>
      <div v-if="requestPending" class="request-section" role="status" aria-label="正在加载关联请求"><h3 v-if="selectedAlertId" class="detail-title">关联请求</h3><el-skeleton :rows="7" animated /></div>
      <div v-else-if="requestError" class="request-section detail-error" role="alert"><p>{{ requestError }}</p><el-button @click="loadSelection">重新加载请求</el-button></div>
      <div v-else-if="requestDetail" :class="{ 'request-section': selectedAlertId }">
        <h3 v-if="selectedAlertId" class="detail-title">关联请求</h3>
        <div v-if="!selectedAlertId && requestDetail.alertIds?.length" class="related-alerts"><span>关联告警</span><el-button v-for="id in requestDetail.alertIds" :key="id" link type="primary" @click="selectAlert(id)">查看告警 {{ id }}</el-button></div>
        <RequestDetail :request="requestDetail" />
      </div>
      <p v-else-if="alertDetail && !alertPending" class="request-section muted">此告警未关联原始请求，无法查看请求处理记录。</p>
    </section>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getTopology, getAlerts, getAlertDetail, getRequestDetail, getAuditRequests } from '@/api/audit'
import { useResource } from '@/composables/useResource'
import HashTopology from '@/components/HashTopology.vue'
import PageHeading from '@/components/PageHeading.vue'
import RequestDetail from '@/components/RequestDetail.vue'
import AccessScope from '@/components/AccessScope.vue'

const route = useRoute()
const router = useRouter()
const detailPanel = ref()
const alertDetail = ref(null)
const requestDetail = ref(null)
const alertPending = ref(false)
const requestPending = ref(false)
const alertError = ref('')
const requestError = ref('')
const selectedAlertId = computed(() => typeof route.query.alertId === 'string' ? route.query.alertId : '')
const selectedRequestId = computed(() => typeof route.query.requestId === 'string' ? route.query.requestId : '')
const selectionActive = computed(() => !!(selectedAlertId.value || selectedRequestId.value))
let selectionVersion = 0

function selectAlert(id) { router.push({ query: { alertId: String(id) } }) }
function selectRequest(requestId) { router.push({ query: { requestId } }) }
function closeDetails() { router.replace({ query: {} }) }

async function loadSelection() {
  const version = ++selectionVersion
  const alertId = selectedAlertId.value
  let requestId = selectedRequestId.value
  alertDetail.value = null
  requestDetail.value = null
  alertError.value = ''
  requestError.value = ''
  alertPending.value = !!alertId
  requestPending.value = !!requestId
  if (!alertId && !requestId) return
  await nextTick()
  if (version !== selectionVersion) return
  detailPanel.value?.scrollIntoView({ block: 'start' })
  detailPanel.value?.focus({ preventScroll: true })
  if (alertId) {
    try {
      const response = await getAlertDetail(alertId)
      if (version !== selectionVersion) return
      if (!response?.data || String(response.data.id) !== alertId) throw new Error('服务返回的告警详情不完整')
      alertDetail.value = response.data
      requestId = response.data.requestId || ''
    } catch (reason) {
      if (version !== selectionVersion) return
      alertError.value = reason.message || '告警详情暂时无法加载'
    } finally {
      if (version === selectionVersion) alertPending.value = false
    }
  }
  if (version !== selectionVersion) return
  requestPending.value = !!requestId
  if (!requestId) return
  try {
    const response = await getRequestDetail(requestId)
    if (version !== selectionVersion) return
    if (!response?.data || response.data.requestId !== requestId) throw new Error('服务返回的请求详情不完整')
    requestDetail.value = response.data
  } catch (reason) {
    if (version === selectionVersion) requestError.value = reason.message || '请求详情暂时无法加载'
  } finally {
    if (version === selectionVersion) requestPending.value = false
  }
}

watch([selectedAlertId, selectedRequestId], loadSelection, { immediate: true })
onBeforeUnmount(() => { selectionVersion += 1 })

const { data, pending, loading, error, updatedAt, hasData, refresh } = useResource(async () => {
  const [topology, alerts, requests] = await Promise.all([getTopology(), getAlerts(), getAuditRequests()])
  if (!topology?.data || !Array.isArray(topology.data.nodes) || !Array.isArray(alerts?.data) || !Array.isArray(requests?.data)) {
    throw new Error('服务返回的审计数据格式不完整')
  }
  return { topology: topology.data, alerts: alerts.data, requests: requests.data }
})
const search = ref('')
const topology = computed(() => data.value?.topology)
const nodes = computed(() => topology.value?.nodes || [])
const updateLabel = computed(() => updatedAt.value?.toLocaleTimeString('zh-CN', { hour12: false }))
const chainState = computed(() => {
  if (!nodes.value.length) return { tone: 'warning', title: '暂无节点，尚无法判断链路状态', description: '刷新对账以获取节点记录。' }
  if (topology.value?.chainStatus === 'inconsistent') return { tone: 'danger', title: '检测到 Hash 不一致', description: '请核对异常节点与链上记录。' }
  if (topology.value?.chainStatus === 'consistent') return { tone: 'success', title: '节点 Hash 一致，链路正常', description: '当前节点指纹与链上记录一致。' }
  return { tone: 'warning', title: '链路状态尚未确认', description: '请刷新对账以获取有效状态。' }
})
const filteredAlerts = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return (data.value?.alerts || []).filter(alert => !keyword || [alert.time, alert.node, typeLabel(alert.type), alert.message, alert.requestId, alert.id].some(value => String(value || '').toLowerCase().includes(keyword)))
})
function statusLabel(status) { return { normal: '正常', tampered: '篡改', unknown: '未知' }[status] || '未知' }
function statusType(status) { return { normal: 'success', tampered: 'danger' }[status] || 'info' }
function typeLabel(type) { return { 'hash-mismatch': 'Hash 不一致', privilege: '越权', jailbreak: '注入攻击' }[type] || type }
function typeTag(type) { return { 'hash-mismatch': 'danger', privilege: 'warning', jailbreak: 'warning' }[type] || 'info' }
function severityLabel(severity) { return { critical: '严重', high: '高', medium: '中', low: '低' }[severity] || '未返回' }
function alertStatusLabel(status) { return { open: '待处理', blocked: '已拦截', resolved: '已处理', acknowledged: '已确认', confirmed: '已确认风险', dismissed: '已排除风险' }[status] || '未返回' }
async function copyHash(hash) {
  try {
    await navigator.clipboard.writeText(hash)
    ElMessage.success('Hash 已复制')
  } catch {
    ElMessage.warning('无法自动复制，请选中 Hash 文本手动复制')
  }
}
</script>

<style scoped>
.chain-status { display: flex; gap: 14px; align-items: center; padding: 20px 24px; border-radius: var(--radius); background: var(--warning-light); color: var(--warning); }
.chain-status > .el-icon { font-size: 24px; flex-shrink: 0; }
.chain-status > div { min-width: 0; }
.chain-status h2 { margin: 0 0 4px; font-size: 15px; font-weight: 600; }
.chain-status p { margin: 0; font-size: 12px; overflow-wrap: anywhere; }
.chain-status.is-danger { color: var(--danger); background: var(--danger-light); }
.chain-status.is-success { color: var(--success); background: var(--success-light); }
.node-count { margin-left: auto; white-space: nowrap; font-size: 12px; }
.audit-grid { display: grid; grid-template-columns: minmax(0, 1.6fr) minmax(0, 1fr); gap: 22px; }
.topology-legend { display: flex; flex-wrap: wrap; justify-content: center; gap: 22px; color: var(--text-sub); font-size: 11px; }
.topology-legend > span { display: inline-flex; align-items: center; gap: 6px; }
.normal { color: var(--success); }
.tampered { color: var(--danger); }
.hash-row { padding: 16px 0; }
.hash-row + .hash-row { border-top: 1px solid var(--border); }
.hash-name { display: flex; align-items: center; justify-content: space-between; gap: 10px; font-size: 13px; font-weight: 600; }
.hash-line { display: flex; align-items: flex-start; gap: 8px; margin-top: 10px; }
.hash-value { font-family: var(--font-data); font-size: 11px; color: var(--text-sub); overflow-wrap: anywhere; flex: 1; padding-top: 5px; }
.tampered .hash-name, .tampered .hash-value { color: var(--danger); }
.copy-button { padding: 4px; min-height: 32px; width: 32px; flex-shrink: 0; color: var(--text-sub); }
.alert-search { max-width: 270px; }
.alert-time { font-size: 11px; }
.request-id-button { display: block; max-width: 100%; font-size: 11px; text-align: left; }
.request-id-button :deep(span) { white-space: normal; overflow-wrap: anywhere; }
.detail-panel { scroll-margin-top: 90px; }
.detail-panel:focus { outline: 2px solid var(--brand); outline-offset: 3px; }
.alert-facts { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 18px 24px; margin: 0 0 24px; }
.alert-facts dt { color: var(--text-sub); font-size: 12px; margin-bottom: 4px; }
.alert-facts dd { margin: 0; font-size: 13px; overflow-wrap: anywhere; }
.detail-title { font-size: 14px; font-weight: 600; margin: 22px 0 10px; }
.detail-copy, .detail-evidence { font-size: 13px; white-space: pre-wrap; overflow-wrap: anywhere; margin: 0; }
.detail-evidence { padding: 14px 16px; background: var(--surface-sub); border-radius: 8px; }
.request-section { border-top: 1px solid var(--border); padding-top: 20px; margin-top: 24px; }
.request-section > .detail-title { margin-top: 0; margin-bottom: 18px; }
.related-alerts { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; margin-bottom: 20px; font-size: 13px; }
.detail-error { color: var(--danger); }
.detail-error p { overflow-wrap: anywhere; }
.page-actions .el-button { gap: 6px; }
@media (max-width: 1100px) { .audit-grid { grid-template-columns: minmax(0, 1fr); } }
@media (max-width: 640px) { .chain-status { padding: 18px 16px; align-items: flex-start; } .node-count { display: none; } .alert-search { max-width: none; } }
@media (max-width: 640px) { .alert-facts { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
