import { computed, ref } from 'vue'

const systemTheme = window.matchMedia('(prefers-color-scheme: dark)')
let preference = null
try {
  const saved = localStorage.getItem('chainwise-theme')
  if (saved === 'light' || saved === 'dark') preference = saved
} catch { /* Theme remains usable when browser storage is unavailable. */ }

const theme = ref(preference || (systemTheme.matches ? 'dark' : 'light'))

function applyTheme() {
  document.documentElement.dataset.theme = theme.value
  document.documentElement.style.colorScheme = theme.value
}

applyTheme()
systemTheme.addEventListener('change', (event) => {
  if (preference) return
  theme.value = event.matches ? 'dark' : 'light'
  applyTheme()
})

const palettes = {
  light: { text: '#24354c', muted: '#627289', border: '#e5eaf1', surface: '#ffffff', brand: '#2463d4', danger: '#c4434d', success: '#20806b', warning: '#a16a13' },
  dark: { text: '#e4eaf3', muted: '#a3b1c6', border: '#2c3a4d', surface: '#182435', brand: '#83afff', danger: '#f08a94', success: '#6ed2b5', warning: '#e6ba68' }
}

export function useTheme() {
  const isDark = computed(() => theme.value === 'dark')
  const chartColors = computed(() => palettes[theme.value])

  function toggleTheme() {
    theme.value = isDark.value ? 'light' : 'dark'
    preference = theme.value
    try { localStorage.setItem('chainwise-theme', preference) } catch { /* Optional persistence. */ }
    applyTheme()
  }

  return { theme, isDark, chartColors, toggleTheme }
}
