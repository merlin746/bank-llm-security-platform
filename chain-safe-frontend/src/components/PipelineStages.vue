<template>
  <div class="pipeline">
    <div v-for="(stage, i) in stages" :key="stage.key" class="stage-row">
      <div class="stage-node" :class="`is-${stage.status}`">
        <el-icon :size="18">
          <CircleCheck v-if="stage.status === 'pass'" />
          <CircleClose v-else-if="stage.status === 'block'" />
          <Remove v-else />
        </el-icon>
        <div class="stage-info">
          <div class="stage-name">
            {{ stage.name }}
            <el-tag size="small" :type="ownerType(stage.owner)" effect="plain">
              {{ ownerLabel(stage.owner) }}
            </el-tag>
          </div>
          <div class="stage-msg" :class="stage.status === 'block' ? 'msg-block' : ''">
            {{ stage.message || '-' }}
          </div>
        </div>
        <div class="stage-latency">{{ stage.latencyMs }}ms</div>
      </div>
      <div v-if="i < stages.length - 1" class="stage-connector" :class="{ broken: stage.status !== 'pass' }">
        <el-icon><Bottom /></el-icon>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  stages: { type: Array, default: () => [] }
})

// 各阶段提供方标签：B=网关 / A=Redis策略(链上) / C=AI服务
function ownerLabel(owner) {
  return { B: '网关', A: '链上策略', C: 'AI 服务', '-': '节点' }[owner] || owner
}
function ownerType(owner) {
  return { B: 'primary', A: 'success', C: 'warning', '-': 'info' }[owner] || 'info'
}
</script>

<style scoped>
.pipeline {
  display: flex;
  flex-direction: column;
}
.stage-row {
  display: flex;
  flex-direction: column;
  align-items: center;
}
.stage-node {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 14px 16px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  background: #fff;
}
.stage-node.is-pass {
  border-left: 4px solid #67c23a;
}
.stage-node.is-pass .el-icon {
  color: #67c23a;
}
.stage-node.is-block {
  border-left: 4px solid #f56c6c;
  background: #fef0f0;
}
.stage-node.is-block .el-icon {
  color: #f56c6c;
}
.stage-node.is-skip {
  border-left: 4px solid #c0c4cc;
  background: #fafafa;
  opacity: 0.6;
}
.stage-node.is-skip .el-icon {
  color: #c0c4cc;
}
.stage-info {
  flex: 1;
}
.stage-name {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}
.stage-msg {
  font-size: 13px;
  color: #909399;
  margin-top: 4px;
}
.stage-msg.msg-block {
  color: #f56c6c;
}
.stage-latency {
  font-size: 12px;
  color: #909399;
  font-variant-numeric: tabular-nums;
}
.stage-connector {
  height: 22px;
  color: #67c23a;
}
.stage-connector.broken {
  color: #c0c4cc;
}
</style>
