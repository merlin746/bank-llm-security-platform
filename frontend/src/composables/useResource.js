import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'

// Keep the last successful snapshot during a failed refresh; never present a failure as zero data.
export function useResource(fetcher, { pollInterval = 0 } = {}) {
  const data = shallowRef(null)
  const pending = ref(false)
  const error = ref('')
  const updatedAt = ref(null)
  const hasData = computed(() => data.value !== null)
  const loading = computed(() => pending.value && !hasData.value)
  let active = true
  let timer

  async function refresh() {
    if (pending.value || !active) return
    pending.value = true
    try {
      const snapshot = await fetcher()
      if (!active) return
      data.value = snapshot
      error.value = ''
      updatedAt.value = new Date()
    } catch (reason) {
      if (active) error.value = reason.message || '数据暂时无法加载，请重试'
    } finally {
      if (active) pending.value = false
    }
  }

  onMounted(() => {
    refresh()
    if (pollInterval) {
      timer = setInterval(() => {
        if (document.visibilityState === 'visible') refresh()
      }, pollInterval)
    }
  })
  onBeforeUnmount(() => {
    active = false
    clearInterval(timer)
  })

  return { data, pending, error, updatedAt, hasData, loading, refresh }
}
