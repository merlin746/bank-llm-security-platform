<template>
  <div class="stat-card" :class="'tone-' + tone">
    <div class="stat-label">{{ label }}</div>
    <div class="stat-value data-number">{{ value }}</div>
    <div class="stat-caption">{{ caption }}</div>
  </div>
</template>

<script setup>
defineProps({
  value: { type: [String, Number], required: true },
  label: { type: String, required: true },
  caption: { type: String, default: '' },
  tone: { type: String, default: 'neutral' }
})
</script>

<style scoped>
.stat-card { padding: 24px 26px; min-width: 0; }
.stat-label { color: var(--text-sub); font-size: 13px; font-weight: 500; }
.stat-value { font-size: 32px; font-weight: 600; line-height: 1.4; letter-spacing: -0.03em; margin-top: 10px; }
.stat-caption { color: var(--text-sub); font-size: 11px; margin-top: 8px; }
.tone-brand .stat-value { color: var(--brand); }
.tone-danger .stat-value { color: var(--danger); }
@media (max-width: 640px) { .stat-card { padding: 20px 16px; } .stat-value { font-size: 27px; } }
</style>
