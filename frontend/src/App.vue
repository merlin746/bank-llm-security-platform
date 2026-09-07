<script setup>
import { ref } from 'vue'
import AttackTest from './views/AttackTest.vue'
import Dashboard from './views/Dashboard.vue'
import AuditTrail from './views/AuditTrail.vue'

const view = ref('attack')
const tabs = [
  { key: 'attack', label: '攻防演练' },
  { key: 'dashboard', label: '安全态势' },
  { key: 'audit', label: '审计溯源' },
]
</script>

<template>
  <header class="topbar">
    <div class="brand">链安智御<small>ChainWise Guard</small><span class="sub">银行大模型全链路安全管控平台 · Demo</span></div>
    <nav class="tabs">
      <div
        v-for="t in tabs"
        :key="t.key"
        class="tab"
        :class="{ active: view === t.key }"
        @click="view = t.key"
      >
        {{ t.label }}
      </div>
    </nav>
  </header>

  <main class="page">
    <AttackTest v-if="view === 'attack'" />
    <Dashboard v-else-if="view === 'dashboard'" />
    <AuditTrail v-else />
  </main>
</template>
