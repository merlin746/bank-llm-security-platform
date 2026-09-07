<script setup>
import { onMounted, onBeforeUnmount, ref } from 'vue'
import * as echarts from 'echarts'
import { api } from '../api'

const list = ref([])
const selected = ref(null)
const error = ref('')
let chart = null

async function load() {
  try {
    const page = await api.requests()
    list.value = page.items || []
  } catch (e) {
    error.value = String(e && e.message ? e.message : e)
  }
}

async function open(row) {
  try {
    selected.value = row
    const detail = await api.requestDetail(row.request_id)
    selected.value = detail.reconciliation || detail
    renderGraph()
  } catch (e) {
    selected.value = row
  }
}

const NODES = [
  { id: 0, name: 'ACCESS 访问' },
  { id: 1, name: 'RAG 检索' },
  { id: 2, name: 'INFERENCE 推理' },
  { id: 3, name: 'DATA_WAREHOUSE 数仓' },
]

function renderGraph() {
  const el = document.getElementById('flow-chart')
  if (!el) return
  chart = echarts.init(el)
  const anomalies = new Set((selected.value.anomalous_nodes || []).map((n) => (typeof n === 'number' ? n : NODES.findIndex((x) => x.name.startsWith(n)))))
  const consistent = selected.value.consistent !== false
  chart.setOption({
    backgroundColor: 'transparent',
    tooltip: {},
    animationDurationUpdate: 600,
    series: [{
      type: 'graph',
      layout: 'none',
      roam: false,
      data: NODES.map((n) => ({
        name: n.name,
        x: n.id * 190,
        y: 60,
        symbolSize: 44,
        itemStyle: { color: anomalies.has(n.id) || (!consistent && n.id === anomalies.values().next().value) ? '#ff5470' : '#2fe6a5' },
        label: { show: true, position: 'bottom', color: '#dbe7ff', fontSize: 11 },
      })),
      links: [
        { source: NODES[0].name, target: NODES[1].name },
        { source: NODES[1].name, target: NODES[2].name },
        { source: NODES[2].name, target: NODES[3].name },
      ],
      lineStyle: { color: '#24344f', width: 2 },
      emphasis: { focus: 'adjacency' },
    }],
  })
}

onMounted(() => {
  load()
})
onBeforeUnmount(() => chart && chart.dispose())
</script>

<template>
  <div class="grid">
    <section class="card">
      <h3>对账请求列表（链上四节点 Hash 对账记录）</h3>
      <table>
        <thead>
          <tr><th>请求 ID</th><th>状态</th><th>异常节点</th><th>共识 Hash</th></tr>
        </thead>
        <tbody>
          <tr v-for="row in list" :key="row.request_id" class="row" @click="open(row)">
            <td class="mono">{{ row.request_id }}</td>
            <td>
              <span class="tag" :class="row.consistent === false ? 'attack' : 'safe'">
                {{ row.consistent === false ? '不一致' : '一致' }}
              </span>
            </td>
            <td>{{ (row.anomalous_nodes || []).join('、') || '—' }}</td>
            <td class="mono">{{ row.consensus_hash }}</td>
          </tr>
        </tbody>
      </table>
      <p v-if="error" class="danger mt">{{ error }}</p>
      <p v-if="!list.length && !error" class="muted mt">暂无数据，请确认 Go 网关已启动。</p>
    </section>

    <section v-if="selected" class="card">
      <h3>四节点 Hash 拓扑 · {{ selected.request_id }}</h3>
      <div id="flow-chart" class="chart"></div>
      <div class="flow-note">
        结果：<span :class="selected.consistent === false ? 'danger' : 'ok-text'">
          {{ selected.consistent === false ? '检测到篡改 / 离群节点' : '四节点数据一致' }}
        </span>
        <span v-if="selected.anomalous_nodes && selected.anomalous_nodes.length">
          ｜异常节点：<span class="danger">{{ selected.anomalous_nodes.join('、') }}</span>
        </span>
        ｜共识 Hash：<span class="mono">{{ selected.consensus_hash }}</span>
      </div>
    </section>
  </div>
</template>
