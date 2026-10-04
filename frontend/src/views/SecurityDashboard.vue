<template>
  <div class="page-container">
    <PageHeading title="授权安全态势" description="查看授权业务范围内的请求流量、拦截趋势与风险行为。">
      <span v-if="updatedAt" class="updated-time">更新于 {{ updateLabel }}</span>
      <el-button :loading="pending" @click="refresh"><el-icon v-if="!pending"><Refresh /></el-icon><span>刷新数据</span></el-button>
    </PageHeading>
    <AccessScope />

    <el-alert v-if="error" class="resource-error" :title="hasData ? '刷新失败，当前展示上次成功加载的数据' : '态势数据加载失败'"
      :description="error + '。请检查服务连接后点击刷新数据。'" type="error" show-icon :closable="false" />

    <section v-if="loading" class="card" aria-label="正在加载安全态势" aria-busy="true">
      <el-skeleton :rows="12" animated />
    </section>
    <template v-else-if="hasData">
      <section class="metrics" aria-label="今日安全指标">
        <StatCard v-for="card in kpiCards" :key="card.label" v-bind="card" />
      </section>

      <div class="chart-grid section-gap">
        <section class="card">
          <div class="panel-heading">
            <div><h2 class="panel-title">实时拦截量趋势</h2><p class="panel-description">拦截量看左轴，放行量看右轴</p></div>
          </div>
          <BaseChart v-if="data.trend.length" :option="trendOption" height="290px" label="请求流量趋势：拦截量对应左轴，放行量对应右轴" />
          <el-empty v-else description="暂无请求趋势数据" :image-size="80" />
        </section>
        <section class="card">
          <div class="panel-heading">
            <div><h2 class="panel-title">风险类型分布</h2><p class="panel-description">按风险事件数量统计</p></div>
          </div>
          <BaseChart v-if="riskTotal" :option="riskOption" height="200px" :label="'风险事件分布，共 ' + riskTotal + ' 次'" />
          <el-empty v-else description="暂无风险事件" :image-size="80" />
          <ul v-if="riskTotal" class="risk-legend" aria-label="风险事件明细">
            <li v-for="(risk, index) in data.riskDistribution" :key="risk.type">
              <span class="risk-key" :style="{ background: riskColors[index % riskColors.length] }" aria-hidden="true"></span>
              <span class="risk-name">{{ risk.type }}</span>
              <span class="data-number">{{ risk.value }}</span>
              <span class="risk-percent">{{ (risk.value / riskTotal * 100).toFixed(1) }}%</span>
            </li>
          </ul>
        </section>
      </div>

      <section class="card section-gap">
        <div class="panel-heading">
          <div><h2 class="panel-title">高风险用户榜单</h2><p class="panel-description">展示 {{ filteredUsers.length }} 位用户，评分范围 0 至 100</p></div>
          <div class="table-filters">
            <el-input v-model="search" placeholder="搜索用户或行为" clearable aria-label="搜索高风险用户">
              <template #prefix><el-icon><Search /></el-icon></template>
            </el-input>
            <el-select v-model="riskLevel" aria-label="筛选风险等级">
              <el-option label="全部等级" value="all" /><el-option label="高风险" value="高" /><el-option label="中风险" value="中" />
            </el-select>
          </div>
        </div>
        <div class="table-scroll" role="region" aria-label="高风险用户表，可横向滚动" tabindex="0">
          <el-table :data="filteredUsers" row-key="name" empty-text="没有符合条件的用户">
            <el-table-column prop="name" label="用户" min-width="150"><template #default="{ row }"><span class="data-number user-name">{{ row.name }}</span></template></el-table-column>
            <el-table-column prop="role" label="角色" min-width="130" />
            <el-table-column prop="lastAction" label="最近行为" min-width="240" />
            <el-table-column label="风险评分" width="140">
              <template #default="{ row }"><span class="score data-number" :class="row.riskScore >= 85 ? 'is-high' : 'is-medium'">{{ row.riskScore }}<small> / 100</small></span></template>
            </el-table-column>
            <el-table-column label="风险等级" width="110"><template #default="{ row }"><el-tag :type="row.level === '高' ? 'danger' : row.level === '中' ? 'warning' : 'info'" size="small" effect="light">{{ row.level }}风险</el-tag></template></el-table-column>
          </el-table>
        </div>
        <div class="table-footer"><span>每 5 秒自动刷新，后台标签页暂停更新</span><router-link to="/attack-defense">前往模拟攻防测试 <el-icon><ArrowRight /></el-icon></router-link></div>
      </section>
    </template>
    <div v-else class="card empty-state">
      <el-icon><Warning /></el-icon><h3>暂时无法获取安全态势</h3><p class="empty-description">恢复服务连接后，刷新数据即可继续查看。</p>
      <el-button type="primary" :loading="pending" @click="refresh">重新加载</el-button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { getOverview, getTrend, getRiskDistribution, getHighRiskUsers } from '@/api/stats'
import { useResource } from '@/composables/useResource'
import { useTheme } from '@/composables/useTheme'
import PageHeading from '@/components/PageHeading.vue'
import AccessScope from '@/components/AccessScope.vue'
import StatCard from '@/components/StatCard.vue'
import BaseChart from '@/components/BaseChart.vue'

const { data, pending, loading, error, updatedAt, hasData, refresh } = useResource(async () => {
  const [overview, trend, risks, users] = await Promise.all([getOverview(), getTrend(), getRiskDistribution(), getHighRiskUsers()])
  if (!overview?.data || !Array.isArray(trend?.data) || !Array.isArray(risks?.data) || !Array.isArray(users?.data)) {
    throw new Error('服务返回的数据格式不完整')
  }
  return { overview: overview.data, trend: trend.data, riskDistribution: risks.data, users: users.data }
}, { pollInterval: 5000 })
const { chartColors, isDark } = useTheme()
const search = ref('')
const riskLevel = ref('all')
const updateLabel = computed(() => updatedAt.value?.toLocaleTimeString('zh-CN', { hour12: false }))
const numberLabel = (value) => value == null ? '暂无' : Number(value).toLocaleString('zh-CN')
const kpiCards = computed(() => {
  const overview = data.value?.overview || {}
  return [
    { value: numberLabel(overview.totalRequests), label: '今日总请求', caption: '进入安全网关的请求', tone: 'brand' },
    { value: numberLabel(overview.blockedToday), label: '今日拦截量', caption: '被安全策略阻断的请求', tone: 'danger' },
    { value: overview.blockRate == null ? '暂无' : overview.blockRate + '%', label: '拦截率', caption: '拦截量占总请求的比例' },
    { value: numberLabel(overview.highRiskUsers), label: '高风险用户', caption: '需重点关注的用户' }
  ]
})
const filteredUsers = computed(() => (data.value?.users || []).filter((user) => {
  const keyword = search.value.trim().toLowerCase()
  return (riskLevel.value === 'all' || user.level === riskLevel.value) && (!keyword || [user.name, user.role, user.lastAction].some(value => String(value || '').toLowerCase().includes(keyword)))
}))
const riskTotal = computed(() => (data.value?.riskDistribution || []).reduce((total, risk) => total + Number(risk.value || 0), 0))
const riskColors = computed(() => isDark.value ? ['#83afff', '#75a8bf', '#a2adc1', '#e6ba68', '#f08a94'] : ['#2463d4', '#547e9a', '#8e9bb0', '#a16a13', '#c4434d'])

const trendOption = computed(() => {
  const colors = chartColors.value
  const trend = data.value?.trend || []
  const axis = { axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: colors.muted, fontSize: 11 }, splitLine: { lineStyle: { color: colors.border, type: 'dashed' } } }
  return {
    textStyle: { fontFamily: "'PingFang SC', 'Microsoft YaHei', sans-serif" },
    aria: { enabled: true },
    tooltip: { trigger: 'axis', confine: true, renderMode: 'richText', backgroundColor: colors.surface, borderColor: colors.border, textStyle: { color: colors.text } },
    legend: { data: ['拦截量', '放行量'], right: 0, top: 0, itemWidth: 16, itemHeight: 3, textStyle: { color: colors.muted, fontSize: 11 } },
    grid: { left: 8, right: 8, top: 48, bottom: 10, containLabel: true },
    xAxis: { ...axis, type: 'category', boundaryGap: false, splitLine: { show: false }, data: trend.map(t => t.time) },
    yAxis: [
      { ...axis, type: 'value', minInterval: 1, name: '拦截量', nameTextStyle: { color: colors.danger, align: 'left' } },
      { ...axis, type: 'value', minInterval: 1, name: '放行量', nameTextStyle: { color: colors.brand, align: 'right' }, splitLine: { show: false } }
    ],
    series: [
      { name: '拦截量', type: 'line', yAxisIndex: 0, smooth: 0.25, showSymbol: false, data: trend.map(t => t.blocked), itemStyle: { color: colors.danger }, lineStyle: { width: 2.5 }, areaStyle: { opacity: 0.05 } },
      { name: '放行量', type: 'line', yAxisIndex: 1, smooth: 0.25, showSymbol: false, data: trend.map(t => t.passed), itemStyle: { color: colors.brand }, lineStyle: { width: 2.5 } }
    ]
  }
})

const riskOption = computed(() => ({
  color: riskColors.value,
  aria: { enabled: true },
  tooltip: { trigger: 'item', confine: true, renderMode: 'richText', formatter: '{b}: {c} ({d}%)', backgroundColor: chartColors.value.surface, borderColor: chartColors.value.border, textStyle: { color: chartColors.value.text } },
  title: { text: riskTotal.value.toLocaleString('zh-CN'), subtext: '风险事件', left: 'center', top: '34%', textStyle: { color: chartColors.value.text, fontSize: 24, fontWeight: 600 }, subtextStyle: { color: chartColors.value.muted, fontSize: 11 } },
  series: [{
    type: 'pie', radius: ['58%', '80%'], center: ['50%', '48%'],
    label: { show: false },
    emphasis: { scale: false }, labelLine: { show: false },
    itemStyle: { borderColor: chartColors.value.surface, borderWidth: 3, borderRadius: 4 },
    data: (data.value?.riskDistribution || []).map(risk => ({ name: risk.type, value: risk.value }))
  }]
}))
</script>

<style scoped>
.metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); overflow: hidden; }
.metrics :deep(.stat-card + .stat-card) { border-left: 1px solid var(--border); }
.chart-grid { display: grid; grid-template-columns: minmax(0, 1.65fr) minmax(0, 1fr); gap: 22px; }
.risk-legend { list-style: none; padding: 0; margin: 0; display: grid; gap: 8px; font-size: 12px; }
.risk-legend li { display: flex; align-items: center; gap: 8px; }
.risk-key { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.risk-name { flex: 1; }
.risk-percent { color: var(--text-sub); width: 48px; text-align: right; font-variant-numeric: tabular-nums; }
.table-filters { display: flex; gap: 10px; }
.table-filters .el-input { width: 210px; }
.table-filters .el-select { width: 120px; }
.user-name { font-size: 12px; }
.score { font-weight: 600; font-size: 16px; }
.score small { color: var(--text-sub); font-size: 10px; font-weight: 400; }
.score.is-high { color: var(--danger); }
.score.is-medium { color: var(--warning); }
.table-footer { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 12px; color: var(--text-sub); font-size: 11px; margin-top: 16px; }
.table-footer a { display: inline-flex; align-items: center; gap: 6px; text-decoration: none; }
.page-actions .el-button { gap: 6px; }
@media (max-width: 1100px) { .chart-grid { grid-template-columns: minmax(0, 1fr); } .table-filters { flex-wrap: wrap; } }
@media (max-width: 640px) {
  .metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metrics :deep(.stat-card:nth-child(3)) { border-left: 0; }
  .metrics :deep(.stat-card:nth-child(n+3)) { border-top: 1px solid var(--border); }
  .table-filters { width: 100%; flex-wrap: nowrap; }
  .table-filters .el-input { width: auto; flex: 1; min-width: 0; }
  .table-filters .el-select { width: 112px; }
}
</style>
