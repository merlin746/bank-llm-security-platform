<template>
  <BaseChart :option="option" height="300px" label="四节点链路拓扑；圆形代表普通节点，菱形代表篡改节点。节点状态与完整指纹可在节点 Hash 明细中查看。" />
</template>

<script setup>
import { computed } from 'vue'
import { useTheme } from '@/composables/useTheme'
import BaseChart from './BaseChart.vue'

const props = defineProps({ topology: { type: Object, required: true } })
const { chartColors } = useTheme()
const option = computed(() => {
  const { nodes = [], edges = [] } = props.topology
  const colors = chartColors.value
  return {
    aria: { enabled: true },
    tooltip: {
      trigger: 'item', confine: true, renderMode: 'richText',
      backgroundColor: colors.surface, borderColor: colors.border, textStyle: { color: colors.text },
      formatter: (params) => {
        const node = nodes.find(node => node.id === params.data.id)
        if (params.dataType !== 'node' || !node) return ''
        const state = { normal: '正常', tampered: '篡改', unknown: '未知' }[node.status] || '未知'
        return node.name + '\nHash: ' + (node.hash || '未返回') + '\n状态: ' + state
      }
    },
    series: [{
      type: 'graph', layout: 'none', symbolSize: 56, roam: true,
      // Let the view preserve the source aspect ratio, including collinear nodes.
      left: 38, right: 38, top: 'middle', scaleLimit: { min: 0.7, max: 2 },
      label: { show: true, position: 'bottom', distance: 12, color: colors.text, fontSize: 11, lineHeight: 18, formatter: (params) => params.name.replace('节点', '\n节点') },
      data: nodes.map((node, index) => {
        const color = node.status === 'tampered' ? colors.danger : node.status === 'normal' ? colors.success : colors.muted
        return {
          id: node.id, name: node.name, x: node.x ?? index * 200, y: node.y ?? 200,
          symbol: node.status === 'tampered' ? 'diamond' : 'circle',
          itemStyle: { color: colors.surface, borderColor: color, borderWidth: 2 },
          emphasis: { itemStyle: { color: colors.surface, borderColor: color, borderWidth: 3 } }
        }
      }),
      links: edges.map(edge => ({ source: edge.from, target: edge.to })),
      lineStyle: { color: colors.muted, width: 1.5, curveness: 0.04, opacity: 0.6 },
      edgeSymbol: ['none', 'arrow'], edgeSymbolSize: [0, 8]
    }],
    media: [{ query: { maxWidth: 440 }, option: { series: [{ symbolSize: 42, left: 28, right: 28, label: { fontSize: 10 } }] } }]
  }
})
</script>
