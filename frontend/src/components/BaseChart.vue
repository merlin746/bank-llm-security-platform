<template>
  <div ref="chartRef" class="base-chart" role="img" :aria-label="label" :style="{ width: '100%', height }"></div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import * as echarts from 'echarts/core'
import { LineChart, PieChart, GraphChart } from 'echarts/charts'
import { AriaComponent, TooltipComponent, LegendComponent, GridComponent, TitleComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([LineChart, PieChart, GraphChart, AriaComponent, TooltipComponent, LegendComponent, GridComponent, TitleComponent, CanvasRenderer])

const props = defineProps({
  option: { type: Object, required: true },
  height: { type: String, default: '320px' },
  label: { type: String, required: true }
})
const chartRef = ref(null)
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)')
let chart
let observer

function render() {
  if (!chartRef.value || !chart) return
  chart.setOption({ ...props.option, animation: !reducedMotion.matches, animationDuration: 250, animationDurationUpdate: 200 }, { notMerge: true })
}

onMounted(() => {
  chart = echarts.init(chartRef.value)
  render()
  observer = new ResizeObserver(() => chart?.resize())
  observer.observe(chartRef.value)
  reducedMotion.addEventListener('change', render)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  reducedMotion.removeEventListener('change', render)
  chart?.dispose()
  chart = null
})
watch(() => props.option, render)
</script>

<style scoped>
.base-chart { min-width: 0; }
</style>
