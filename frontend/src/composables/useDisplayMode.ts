import { ref, onMounted } from 'vue'
import { getConfig } from '@/api'

export type DisplayMode = 'all' | 'linux' | 'windows'

export function useDisplayMode() {
  const displayMode = ref<DisplayMode>('all')
  const ready = ref(false)

  onMounted(async () => {
    try {
      const config = await getConfig()
      const mode = config.display_mode
      displayMode.value = (mode === 'linux' || mode === 'windows') ? mode : 'all'
    } catch {
      displayMode.value = 'all'
    } finally {
      ready.value = true
    }
  })

  return { displayMode, ready }
}
