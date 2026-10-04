<template>
  <div v-if="canShowRequest" class="request-detail">
    <dl class="request-facts">
      <div><dt>请求编号</dt><dd class="data-number">{{ request.requestId }}</dd></div>
      <div><dt>请求时间</dt><dd>{{ request.time || '未返回' }}</dd></div>
      <div><dt>请求类型</dt><dd>{{ kindLabel }}</dd></div>
      <div><dt>最终裁决</dt><dd><el-tag :type="request.verdict === 'block' ? 'danger' : request.verdict === 'pass' ? 'success' : 'info'" size="small">{{ verdictLabel }}</el-tag></dd></div>
      <div><dt>总耗时</dt><dd class="data-number">{{ request.totalLatencyMs ?? '未知' }}<template v-if="request.totalLatencyMs != null"> ms</template></dd></div>
      <div v-if="request.role"><dt>{{ isSimulation ? '测试角色' : '处理角色' }}</dt><dd>{{ request.role }}</dd></div>
      <div v-if="request.dataLevel"><dt>记录密级</dt><dd>{{ request.dataLevel }}</dd></div>
      <div v-if="request.requestedLevel"><dt>测试访问密级</dt><dd>{{ request.requestedLevel }}</dd></div>
      <div v-if="request.action"><dt>操作</dt><dd>{{ request.action }}</dd></div>
    </dl>
    <template v-if="request.result"><h4>业务处理结果</h4><p class="request-input">{{ request.result }}</p></template>
    <template v-if="request.riskTip || request.riskHint"><h4>风险提示</h4><p class="request-input">{{ request.riskTip || request.riskHint }}</p></template>
    <template v-if="canShowTrace && (request.kind === 'attack' || request.prompt != null)">
      <h4>原始输入</h4><p class="request-input">{{ request.prompt || '未返回原始输入' }}</p>
    </template>
    <template v-if="canShowTrace">
      <DetectionNotice v-if="request.detection || request.kind === 'attack'" :detection="request.detection" />
      <h4>逐层处理记录</h4>
      <PipelineStages v-if="request.stages?.length" :stages="request.stages" />
      <p v-else class="muted">此请求未保留逐层处理记录。</p>
    </template>
  </div>
  <el-empty v-else description="当前账号未获此请求的查看授权" :image-size="60" />
</template>

<script setup>
import { computed } from 'vue'
import DetectionNotice from '@/components/DetectionNotice.vue'
import PipelineStages from '@/components/PipelineStages.vue'
import { useUserStore } from '@/store/user'
import { canAccessRecord, hasPermission } from '@/config/permissions'

const props = defineProps({ request: { type: Object, required: true } })
const userStore = useUserStore()
const isSimulation = computed(() => ['attack', 'access'].includes(props.request.kind))
const isOwnSimulation = computed(() => isSimulation.value && props.request.owner === userStore.username && hasPermission(userStore.userInfo, 'simulation.run'))
// The API is the authority. This guard also removes a retained detail as soon
// as the active account changes; simulation target levels are not actual data.
const canShowRequest = computed(() => userStore.isLogin && canAccessRecord(userStore.userInfo, props.request))
const canShowTrace = computed(() => canShowRequest.value && (isOwnSimulation.value || hasPermission(userStore.userInfo, 'audit.read') || hasPermission(userStore.userInfo, 'alerts.read')))
const kindLabel = computed(() => ({ attack: 'Prompt 注入测试', access: '越权访问测试', audit: '链路审计', business: '授权业务' }[props.request.kind] || '未返回'))
const verdictLabel = computed(() => ({ block: '已拦截', pass: '已放行' }[props.request.verdict] || '状态未知'))
</script>

<style scoped>
.request-facts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px 24px; margin: 0 0 24px; }
.request-facts dt { font-size: 12px; color: var(--text-sub); margin-bottom: 4px; }
.request-facts dd { margin: 0; font-size: 13px; overflow-wrap: anywhere; }
h4 { margin: 24px 0 12px; font-size: 14px; font-weight: 600; }
.request-input { white-space: pre-wrap; overflow-wrap: anywhere; background: var(--surface-sub); border-radius: 8px; padding: 14px 16px; margin: 0 0 20px; font-size: 13px; }
@media (max-width: 640px) { .request-facts { grid-template-columns: minmax(0, 1fr); } }
</style>
