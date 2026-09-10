<template>
  <BaseChart :option="option" height="380px" />
</template>

<script setup>
import { computed } from 'vue'
import BaseChart from './BaseChart.vue'

const props = defineProps({
  topology: { type: Object, required: true }
})

const STATUS_STYLE = {
  normal: { color: '#67c23a', border: '#67c23a' },
  tampered: { color: '#f56c6c', border: '#ff0000' },
  unknown: { color: '#909399', border: '#c0c4cc' }
}

const option = computed(() => {
  const { nodes = [], edges = [] } = props.topology
  return {
    tooltip: {
      trigger: 'item',
      formatter: (p) => {
        if (p.dataType === 'node') {
          const n = nodes.find((x) => x.id === p.data.id)
          const stateText = n.status === 'tampered' ? '⚠ 已篡改' : n.status === 'normal' ? '正常' : '未知'
          return `<b>${n.name}</b><br/>Hash: ${n.hash}<br/>状态: ${stateText}`
        }
        return ''
      }
    },
    series: [
      {
        type: 'graph',
        layout: 'none',
        symbolSize: 84,
        roam: true,
        label: { show: true, position: 'bottom', fontSize: 12, formatter: '{b}' },
        data: nodes.map((n) => {
          const st = STATUS_STYLE[n.status] || STATUS_STYLE.unknown
          return {
            id: n.id,
            name: n.name,
            x: n.x,
            y: n.y,
            symbol: n.status === 'tampered' ? 'diamond' : 'circle',
            itemStyle: {
              color: st.color,
              // 篡改节点用红色粗边框（红框标记）
              borderColor: st.border,
              borderWidth: n.status === 'tampered' ? 5 : 2,
              shadowBlur: n.status === 'tampered' ? 20 : 0,
              shadowColor: 'rgba(245,108,108,0.5)'
            }
          }
        }),
        links: edges.map((e) => ({ source: e.from, target: e.to })),
        lineStyle: { color: '#909399', width: 2, curveness: 0.1 },
        edgeSymbol: ['circle', 'arrow'],
        edgeSymbolSize: [5, 10]
      }
    ]
  }
})
</script>
