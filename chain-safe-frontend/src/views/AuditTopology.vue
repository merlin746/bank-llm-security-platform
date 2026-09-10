<template>
  <div class="page-container">
    <h2 class="page-title">审计溯源对账</h2>

    <!-- 链状态提示 -->
    <el-alert
      v-if="topology"
      :title="topology.alert || (topology.chainStatus === 'consistent' ? '四节点 Hash 一致，链路正常' : '检测到 Hash 不一致')"
      :type="topology.chainStatus === 'consistent' ? 'success' : 'error'"
      :closable="false"
      show-icon
      class="mb16"
    />

    <el-row :gutter="16">
      <!-- 四节点 Hash 拓扑 -->
      <el-col :span="16">
        <div class="card">
          <div class="chart-title">四节点 Hash 连贯性拓扑</div>
          <HashTopology v-if="topology" :topology="topology" />
          <el-empty v-else description="加载中..." />
        </div>
      </el-col>

      <!-- 节点 Hash 明细 -->
      <el-col :span="8">
        <div class="card">
          <div class="chart-title">节点 Hash 明细</div>
          <div v-for="n in nodes" :key="n.id" class="hash-row" :class="{ tampered: n.status === 'tampered' }">
            <div class="hash-name">
              {{ n.name }}
              <el-tag v-if="n.status === 'tampered'" type="danger" size="small">篡改</el-tag>
              <el-tag v-else type="success" size="small">正常</el-tag>
            </div>
            <div class="hash-value">{{ n.hash }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- 告警日志 -->
    <div class="card mt16">
      <div class="chart-title">告警日志</div>
      <el-table :data="alerts" stripe>
        <el-table-column prop="time" label="时间" width="200" />
        <el-table-column prop="node" label="节点" width="160" />
        <el-table-column label="类型" width="160">
          <template #default="{ row }">
            <el-tag size="small" :type="typeTag(row.type)">{{ typeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="内容" />
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { getTopology, getAlerts } from '@/api/audit'
import HashTopology from '@/components/HashTopology.vue'

const topology = ref(null)
const alerts = ref([])

const nodes = computed(() => topology.value?.nodes || [])

function typeLabel(type) {
  return { 'hash-mismatch': 'Hash 不一致', privilege: '越权', jailbreak: '注入攻击' }[type] || type
}
function typeTag(type) {
  return { 'hash-mismatch': 'danger', privilege: 'warning', jailbreak: 'warning' }[type] || 'info'
}

async function load() {
  try {
    const [tp, al] = await Promise.all([getTopology(), getAlerts()])
    topology.value = tp.data
    alerts.value = al.data
  } catch (e) {
    console.error('加载审计数据失败', e)
  }
}

onMounted(load)
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
.hash-row {
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}
.hash-row.tampered {
  background: #fef0f0;
  border-radius: 6px;
  padding-left: 8px;
}
.hash-name {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}
.hash-value {
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  word-break: break-all;
}
</style>
