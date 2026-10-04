<template>
  <el-alert class="detection-notice" :title="notice.title" :description="notice.description"
    :type="notice.type" show-icon :closable="false" />
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({ detection: { type: Object, default: null } })
const notice = computed(() => {
  const detection = props.detection
  if (detection?.source === 'mock' || detection?.mode === 'mock') {
    return { type: 'info', title: '演示检测结果', description: '本次使用模拟数据，未调用实际检测服务。' }
  }
  if (!detection || !['rules', 'model'].includes(detection.mode)) {
    return { type: 'warning', title: '检测能力尚未确认', description: detection?.reason || '服务未返回完整的检测来源与模式，请复核本次裁决。' }
  }
  if (detection?.degraded) {
    return {
      type: 'warning', title: '检测已降级为规则匹配',
      description: `${detection.reason || '语义模型暂不可用'}。本次仅使用备用规则，未命中规则的请求仍需复核。`
    }
  }
  return {
    type: 'info', title: detection.mode === 'model' ? '本次使用语义模型检测' : '本次使用规则检测',
    description: detection.reason || (detection.mode === 'model' ? '检测服务已返回模型判定。' : '请求由检测服务的规则层处理。')
  }
})
</script>

<style scoped>
.detection-notice { margin-bottom: 18px; }
</style>
