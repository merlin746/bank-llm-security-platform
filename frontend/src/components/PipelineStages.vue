<template>
  <ol class="pipeline" aria-label="逐层安全检测结果">
    <li v-for="stage in stages" :key="stage.key" class="stage-row" :class="'is-' + stage.status">
      <div class="stage-marker" aria-hidden="true"><el-icon><CircleCheck v-if="stage.status === 'pass'" /><CircleClose v-else-if="stage.status === 'block'" /><Remove v-else /></el-icon></div>
      <div class="stage-info">
        <div class="stage-heading"><span class="stage-name">{{ stage.name }}</span><span class="stage-status">{{ statusLabel(stage.status) }}</span></div>
        <p class="stage-message">{{ stage.message || '未返回说明' }}</p>
        <div class="stage-detail"><span>{{ ownerLabel(stage.owner) }}</span><span class="data-number">{{ stage.latencyMs ?? '未知' }}<template v-if="stage.latencyMs != null"> ms</template></span></div>
      </div>
    </li>
  </ol>
</template>

<script setup>
defineProps({ stages: { type: Array, default: () => [] } })
function ownerLabel(owner) {
  return { B: '网关', A: '链上策略', C: 'AI 服务', '-': '节点' }[owner] || owner || '未知节点'
}
function statusLabel(status) {
  return { pass: '已通过', block: '已拦截', skip: '未执行' }[status] || '状态未知'
}
</script>

<style scoped>
.pipeline { list-style: none; padding: 0; margin: 0; }
.stage-row { position: relative; display: flex; gap: 14px; padding: 16px 0; }
.stage-row:not(:last-child)::after { content: ''; position: absolute; top: 44px; bottom: -16px; left: 13px; width: 1px; background: var(--border); }
.stage-marker { position: relative; display: grid; place-items: center; width: 28px; height: 28px; flex-shrink: 0; font-size: 19px; color: var(--text-sub); background: var(--surface-sub); border-radius: 50%; }
.stage-info { flex: 1; min-width: 0; }
.stage-heading { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.stage-name { font-weight: 600; font-size: 13px; }
.stage-status { font-size: 11px; white-space: nowrap; color: var(--text-sub); }
.stage-message { font-size: 12px; color: var(--text-sub); margin: 4px 0 8px; overflow-wrap: anywhere; }
.stage-detail { display: flex; gap: 16px; font-size: 10px; color: var(--text-sub); }
.is-pass .stage-marker { color: var(--success); background: var(--success-light); }
.is-pass .stage-status { color: var(--success); }
.is-block .stage-marker { color: var(--danger); background: var(--danger-light); }
.is-block .stage-status, .is-block .stage-message { color: var(--danger); }
</style>
