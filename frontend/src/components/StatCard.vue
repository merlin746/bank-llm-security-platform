<template>
  <div class="stat-card">
    <div class="stat-icon" :style="{ background: bg }">
      <el-icon :size="26" color="#fff"><component :is="icon" /></el-icon>
    </div>
    <div class="stat-body">
      <div class="stat-value">{{ value }}</div>
      <div class="stat-label">{{ label }}</div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  icon: { type: String, required: true }, // 图标组件名（已在 main.js 全局注册）
  bg: { type: String, default: '#1e6fff' },
  value: { type: [String, Number], required: true },
  label: { type: String, required: true }
})
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.06);
}
.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.stat-value {
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
}
.stat-label {
  font-size: 13px;
  color: #909399;
  margin-top: 6px;
}
</style>
