import { ref, onMounted, onBeforeUnmount } from 'vue'

export function useFullscreen(onChange?: (isFullscreen: boolean) => void) {
  const isFullscreen = ref(false)

  function toggleFullscreen() {
    document.fullscreenElement
      ? document.exitFullscreen()
      : document.documentElement.requestFullscreen()
  }

  function handleFullscreenChange() {
    isFullscreen.value = !!document.fullscreenElement
    onChange?.(isFullscreen.value)
  }

  onMounted(() => document.addEventListener('fullscreenchange', handleFullscreenChange))
  onBeforeUnmount(() => document.removeEventListener('fullscreenchange', handleFullscreenChange))

  return { isFullscreen, toggleFullscreen }
}
