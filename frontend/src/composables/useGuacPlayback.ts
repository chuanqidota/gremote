import { ref, nextTick, onUnmounted } from 'vue'
import type { Ref } from 'vue'

export function useGuacPlayback() {
  const loading = ref(true)
  const loadingProgress = ref(0)
  const loadingLabel = ref('正在加载录制文件...')
  const error = ref('')
  const paused = ref(true)
  const progress = ref(0)
  const currentTime = ref(0)
  const duration = ref(0)
  const seeking = ref(false)
  const seekProgress = ref(0)

  let recording: any = null
  let containerEl: HTMLDivElement | null = null
  let blobSize = 0
  let loaded = false

  function fitDisplay() {
    if (!recording || !containerEl) return
    const display = recording.getDisplay()
    const containerW = containerEl.clientWidth
    const containerH = containerEl.clientHeight
    const displayW = display.getWidth()
    const displayH = display.getHeight()
    if (displayW === 0 || displayH === 0 || containerW === 0 || containerH === 0) return
    const scale = Math.min(containerW / displayW, containerH / displayH)
    display.scale(scale)
  }

  function setContainer(el: HTMLDivElement) {
    containerEl = el
  }

  async function load(url: string, containerOrGetter?: HTMLDivElement | null | (() => HTMLDivElement | null)) {
    const getContainer = typeof containerOrGetter === 'function'
      ? containerOrGetter
      : () => containerOrGetter ?? null
    const { default: Guacamole } = await import('guacamole-common-js')

    let response: Response
    try {
      response = await fetch(url)
    } catch (e: any) {
      throw new Error(`网络请求失败: ${e?.message || e}`)
    }

    if (!response.ok) {
      let msg = `HTTP ${response.status}: ${response.statusText}`
      try {
        const errBody = await response.text()
        if (errBody) msg += ` - ${errBody}`
      } catch {}
      throw new Error(msg)
    }

    const blob = await response.blob()
    blobSize = blob.size
    if (blobSize === 0) {
      throw new Error('录制文件为空，可能录制未正常完成')
    }

    loadingLabel.value = '正在解析录制数据...'

    console.log('[GuacPlayback] Creating SessionRecording, blob size:', blobSize)
    const rec = new Guacamole.SessionRecording(blob)

    rec.onerror = (msg: string) => {
      console.error('[GuacPlayback] onerror:', msg)
      error.value = `回放错误: ${msg}`
      loading.value = false
    }

    rec.onprogress = (dur: number, parsedSize: number) => {
      console.log('[GuacPlayback] onprogress: dur=', dur, 'parsed=', parsedSize, 'blobSize=', blobSize)
      if (blobSize > 0) {
        loadingProgress.value = Math.min(parsedSize / blobSize, 1)
      }
      duration.value = rec.getDuration() / 1000
    }

    rec.onload = () => {
      console.log('[GuacPlayback] onload: duration=', rec.getDuration())
      duration.value = rec.getDuration() / 1000
      if (duration.value === 0) {
        error.value = '录制文件格式无效，未找到有效的录制帧'
        loading.value = false
        return
      }
      loaded = true
      loading.value = false
      loadingProgress.value = 1

      // Wait for Vue to render the player container after loading becomes false
      nextTick(() => {
        const display = rec.getDisplay()
        const displayEl = display.getElement()
        displayEl.style.cursor = 'none'
        const target = containerEl || getContainer()
        console.log('[GuacPlayback] target:', target, 'displayW:', display.getWidth(), 'displayH:', display.getHeight())
        if (target) {
          target.appendChild(displayEl)
          containerEl = target
          console.log('[GuacPlayback] appended, containerW:', target.clientWidth, 'containerH:', target.clientHeight)
        } else {
          console.warn('[GuacPlayback] containerEl is null, display not appended')
        }

        display.onresize = (w, h) => { console.log('[GuacPlayback] onresize:', w, h); fitDisplay() }
        fitDisplay()
      })
    }

    rec.onplay = () => { paused.value = false }
    rec.onpause = () => { paused.value = true }
    rec.onseek = (position: number, current: number, total: number) => {
      currentTime.value = position / 1000
      progress.value = duration.value > 0 ? (position / 1000 / duration.value) * 100 : 0
      if (total > 1) {
        seeking.value = true
        seekProgress.value = current / total
      } else {
        seeking.value = false
      }
    }

    recording = rec
  }

  function play() {
    recording?.play()
  }

  function pause() {
    recording?.pause()
  }

  function togglePlay() {
    if (!recording) return
    if (recording.isPlaying()) {
      recording.pause()
    } else {
      recording.play()
    }
  }

  function seek(percentage: number) {
    if (!recording || duration.value <= 0) return
    const positionMs = (percentage / 100) * duration.value * 1000
    recording.seek(positionMs)
  }

  function destroy() {
    if (recording) {
      recording.abort()
      const displayEl = recording.getDisplay()?.getElement()
      if (displayEl && displayEl.parentNode) {
        displayEl.parentNode.removeChild(displayEl)
      }
      recording = null
    }
    loaded = false
    containerEl = null
  }

  onUnmounted(destroy)

  return {
    loading,
    loadingProgress,
    loadingLabel,
    error,
    paused,
    progress,
    currentTime,
    duration,
    seeking,
    seekProgress,
    load,
    play,
    pause,
    togglePlay,
    seek,
    fitDisplay,
    setContainer,
    destroy,
  }
}
