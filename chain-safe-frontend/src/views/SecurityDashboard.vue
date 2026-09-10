<template>
  <div class="page-container">
    <h2 class="page-title">安全态势大屏</h2>

    <!-- KPI 卡片 -->
    <el-row :gutter="16" class="mb16">
      <el-col v-for="card in kpiCards" :key="card.label" :span="6">
        <StatCard :icon="card.icon" :bg="card.bg" :value="card.value" :label="card.label" />
      </el-col>
    </el-row>

    <!-- 图表区 -->
    <el-row :gutter="16">
      <el-col :span="15">
        <div class="card">
          <div class="chart-title">实时拦截量趋势</div>
          <BaseChart :option="trendOption" height="340px" />
        </div>
      </el-col>
      <el-col :span="9">
        <div class="card">
          <div class="chart-title">风险类型分布</div>
          <BaseChart :option="riskOption" height="340px" />
        </div>
      </el-col>
    </el-row>

    <!-- 高风险用户榜单 -->
    <div class="card mt16">
      <div class="chart-title">高风险用户榜单</div>
      <el-table :data="highRiskUsers" stripe>
        <el-table-column prop="name" label="用户" width="160" />
        <el-table-column prop="role" label="角色" width="140" />
        <el-table-column prop="lastAction" label="最近行为" />
        <el-table-column label="风险评分" width="200">
          <template #default="{ row }">
            <el-progress
              :percentage="row.riskScore"
              :color="riskColor"
              :stroke-width="10"
            />
          </template>
        </el-table-column>
        <el-table-column prop="level" label="风险等级" width="110">
          <template #default="{ row }">
            <el-tag :type="row.level === '高' ? 'danger' : 'warning'" size="small">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { getOverview, getTrend, getRiskDistribution, getHighRiskUsers } from '@/api/stats'
import StatCard from '@/components/StatCard.vue'
import BaseChart from '@/components/BaseChart.vue'

const overview = ref({ totalRequests: 0, blockedToday: 0, blockRate: 0, highRiskUsers: 0 })
const trend = ref([])
const riskDistribution = ref([])
const highRiskUsers = ref([])

const kpiCards = computed(() => [
  { icon: 'DataLine', bg: '#1e6fff', value: overview.value.totalRequests.toLocaleString(), label: '今日总请求' },
  { icon: 'Warning', bg: '#f56c6c', value: overview.value.blockedToday, label: '今日拦截量' },
  { icon: 'TrendCharts', bg: '#e6a23c', value: `${overview.value.blockRate}%`, label: '拦截率' },
  { icon: 'User', bg: '#909399', value: overview.value.highRiskUsers, label: '高风险用户' }
])

const trendOption = computed(() => ({
  tooltip: { trigger: 'axis' },
  legend: { data: ['拦截量', '放行量'] },
  grid: { left: 40, right: 16, top: 40, bottom: 30 },
  xAxis: { type: 'category', data: trend.value.map((t) => t.time) },
  yAxis: { type: 'value' },
  series: [
    { name: '拦截量', type: 'line', smooth: true, data: trend.value.map((t) => t.blocked), itemStyle: { color: '#f56c6c' }, areaStyle: { opacity: 0.08 } },
    { name: '放行量', type: 'line', smooth: true, data: trend.value.map((t) => t.passed), itemStyle: { color: '#67c23a' }, areaStyle: { opacity: 0.06 } }
  ]
}))

const riskOption = computed(() => ({
  tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
  legend: { orient: 'vertical', right: 10, top: 'center' },
  series: [
    {
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['40%', '50%'],
      label: { show: false },
      data: riskDistribution.value.map((r) => ({ name: r.type, value: r.value }))
    }
  ]
}))

function riskColor(percentage) {
  return percentage >= 85 ? '#f56c6c' : percentage >= 70 ? '#e6a23c' : '#67c23a'
}

async function loadAll() {
  try {
    const [ov, tr, rk, ur] = await Promise.all([
      getOverview(),
      getTrend(),
      getRiskDistribution(),
      getHighRiskUsers()
    ])
    overview.value = ov.data
    trend.value = tr.data
    riskDistribution.value = rk.data
    highRiskUsers.value = ur.data
  } catch (e) {
    console.error('加载态势数据失败', e)
  }
}

let timer = null
onMounted(() => {
  loadAll()
  // 每 5 秒轮询一次，模拟实时大屏
  timer = setInterval(loadAll, 5000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<style scoped>
.mb16 {
  margin-bottom: 16px;
}
.mt16 {
  margin-top: 16px;
}
.chart-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 12px;
}
</style>
