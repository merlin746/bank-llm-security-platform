<template>
  <div class="brand-logo" :class="{ 'is-inverse': inverse, 'is-compact': compact }" aria-label="链安智御 ChainWise Guard">
    <svg class="brand-mark" viewBox="0 0 48 48" aria-hidden="true" focusable="false">
      <use :href="markSource" />
    </svg>
    <div class="brand-wordmark">
      <span class="brand-title">链安智御</span>
      <span class="brand-subtitle">ChainWise Guard</span>
    </div>
  </div>
</template>

<script setup>
defineProps({ inverse: Boolean, compact: Boolean })
const markSource = import.meta.env.BASE_URL + 'brand-mark.svg#chainwise-mark'
</script>

<style scoped>
.brand-logo { --mark-primary: var(--brand); --mark-secondary: #548dc4; display: inline-flex; align-items: center; gap: 12px; color: var(--text-main); flex-shrink: 0; }
.brand-mark { width: 44px; height: 44px; flex-shrink: 0; }
.brand-wordmark { display: grid; gap: 3px; }
.brand-title { font-size: 23px; font-weight: 700; line-height: 1.3; letter-spacing: 0.055em; white-space: nowrap; }
.brand-subtitle { font-size: 10px; line-height: 1.4; letter-spacing: 0.07em; color: var(--text-sub); }
.is-inverse { --mark-primary: #80aeff; --mark-secondary: #b2d3ea; color: #f0f5ff; }
.is-inverse .brand-subtitle { color: #aebfd5; }
.is-compact .brand-mark { width: 36px; height: 36px; }
.is-compact .brand-title { font-size: 21px; }
.is-compact .brand-subtitle { display: none; }
@media (max-width: 480px) { .brand-mark { width: 38px; height: 38px; } .brand-title { font-size: 21px; } .brand-subtitle { font-size: 9px; } }
</style>
