import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

export type MofangTheme = 'light' | 'dark' | 'system'

export function useMofangAppearance() {
  const media = window.matchMedia('(prefers-color-scheme: dark)')
  const motion = window.matchMedia('(prefers-reduced-motion: reduce)')
  function readTheme(): MofangTheme {
    try {
      const value = localStorage.getItem('theme')
      return value === 'light' || value === 'dark' ? value : 'system'
    } catch {
      return 'system'
    }
  }
  const theme = ref<MofangTheme>(readTheme())
  const systemDark = ref(media.matches)
  const reducedMotion = ref(motion.matches)
  const motionPaused = ref(false)
  const isDark = computed(() => theme.value === 'dark' || (theme.value === 'system' && systemDark.value))
  const animate = computed(() => !reducedMotion.value && !motionPaused.value)

  function syncTheme() {
    document.documentElement.classList.toggle('dark', isDark.value)
  }
  function setTheme(value: MofangTheme) {
    theme.value = value
    try {
      if (value === 'system') localStorage.removeItem('theme')
      else localStorage.setItem('theme', value)
    } catch { /* The current appearance still works when browser storage is unavailable. */ }
    syncTheme()
  }
  function onSystemChange(event: MediaQueryListEvent) {
    systemDark.value = event.matches
    syncTheme()
  }
  function onMotionChange(event: MediaQueryListEvent) {
    reducedMotion.value = event.matches
  }
  function onStorage(event: StorageEvent) {
    if (event.key === 'theme' || event.key === null) {
      theme.value = readTheme()
      syncTheme()
    }
  }
  syncTheme()
  onMounted(() => {
    media.addEventListener('change', onSystemChange)
    motion.addEventListener('change', onMotionChange)
    window.addEventListener('storage', onStorage)
  })
  onBeforeUnmount(() => {
    media.removeEventListener('change', onSystemChange)
    motion.removeEventListener('change', onMotionChange)
    window.removeEventListener('storage', onStorage)
  })
  return { theme, isDark, reducedMotion, motionPaused, animate, setTheme }
}
