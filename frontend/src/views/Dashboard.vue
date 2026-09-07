<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import * as echarts from 'echarts'
import { api } from '../api'

const stats = ref(null)
const anomalies = ref([])
let chart = null

function renderChart() {
  const el = document.getElementById('trend-chart')
  if (!el) return
  chart = echarts.init(el)
  const hours = Array.from({ length: 12 }, (_, i) => (23 - i) + ':00').reverse()
  const intercepted = [2, 5, 3, 8, 6, 12, 9, 14, 10, 18, 13, 20]
  const blocked = [1, 2, 1, 4, 3, 6, 4, 8, 5, 9, 7, 11]
  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis' },
    legend: { data: ['检测请求', '拦截请求'], textStyle: { color: '#8297bd' } },
    grid: { left: 40, right: 20, top: 36, bottom: 24 },
    xAxis: { type: 'category', data: hours, axisLabel: { color: '#8297bd' } },
    yAxis: { type: 'value', axisLabel: { color: '#8297bd' }, splitLine: { lineStyle: { color: '#1a2942' } } },
    series: [
      { name: '检测请求', type: 'line', smooth: true, data: intercepted, itemStyle: { color: '#37d0ff' }, areaStyle: { color: 'rgba(55,208,255,0.08)' } },
      { name: '拦截请求', type: 'line', smooth: true, data: blocked, itemStyle: { color: '#ff5470' } },
    ],
  })
}

onMounted(async () => {
  try {
    stats.value = await api.stats()
    anomalies.value = await api.anomalies()
  } catch { /* 后端未启动时保持空 */ }
  renderChart()
  window.addEventListener('resize', onResize)
})

function onResize() { chart && chart.resize() }
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  chart && chart.dispose()
})
</script>

<template>
  <div class="grid">
    <div class="cards">
      <section class="card">
        <h3>链上对账总记录</h3>
        <div class="stat-num ok">{{ stats ? stats.total_records : '—' }}</div>
      </section>
      <section class="card">
        <h3>异常请求</h3>
        <div class="stat-num bad">{{ stats ? stats.total_anomalies : '—' }}</div>
      </section>
      <section class="card">
        <h3>已完成对账</h3>
        <div class="stat-num warn">{{ stats ? stats.reconciled_count : '—' }}</div>
      </section>
      <section class="card">
        <h3>实时告警（最近异常）</h3>
        <div v-if="anomalies.length">
          <div v-for="a in anomalies" :key="a.request_id" class="kv" style="grid-template-columns:1fr">
            <dd><span class="tag attack">{{ a.node_type }}</span> <span class="mono">{{ a.request_id }}</span></dd>
          </div>
        </div>
        <div v-else class="muted">暂无告警</div>
      </section>
    </div>

    <section class="card">
      <h3>拦截趋势（近 12 小时 · 演示数据）</h3>
      <div id="trend-chart" class="chart"></div>
    </section>
  </div>
</template>
