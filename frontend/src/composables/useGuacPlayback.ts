import { ref, nextTick, onUnmounted } from 'vue'

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
  const playbackSpeed = ref(1)
  const ended = ref(false)

  let recording: any = null
  let containerEl: HTMLDivElement | null = null
  let blobSize = 0
  let loaded = false
  let speedTimer: ReturnType<typeof setInterval> | null = null
  const SPEED_TICK_MS = 200

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

    const rec = new Guacamole.SessionRecording(blob)

    rec.onerror = (msg: string) => {
      error.value = `回放错误: ${msg}`
      loading.value = false
    }

    rec.onprogress = (dur: number, parsedSize: number) => {
      if (blobSize > 0) {
        loadingProgress.value = Math.min(parsedSize / blobSize, 1)
      }
      duration.value = rec.getDuration() / 1000
    }

    rec.onload = () => {
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
        if (target) {
          target.appendChild(displayEl)
          containerEl = target
        }

        display.onresize = () => fitDisplay()
        fitDisplay()
      })
    }

    rec.onplay = () => { paused.value = false; ended.value = false }
    rec.onpause = () => {
      paused.value = true
      stopSpeedPlayback()
      if (rec.getPosition() >= rec.getDuration() && rec.getDuration() > 0) {
        ended.value = true
      }
    }
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
    if (!recording) return
    if (playbackSpeed.value === 1) {
      recording.play()
    } else {
      startSpeedPlayback()
    }
  }

  function pause() {
    stopSpeedPlayback()
    recording?.pause()
  }

  function togglePlay() {
    if (!recording) return
    if (recording.isPlaying() || speedTimer) {
      pause()
    } else {
      if (ended.value) {
        ended.value = false
        recording.seek(0, () => { play() })
      } else {
        play()
      }
    }
  }

  function seek(percentage: number) {
    if (!recording || duration.value <= 0) return
    const wasPlaying = recording.isPlaying() || !!speedTimer
    stopSpeedPlayback()
    const positionMs = (percentage / 100) * duration.value * 1000
    recording.seek(positionMs, () => {
      if (wasPlaying) play()
    })
  }

  function setSpeed(speed: number) {
    playbackSpeed.value = speed
    // If currently playing with custom loop, restart with new speed
    if (speedTimer) {
      stopSpeedPlayback()
      startSpeedPlayback()
    } else if (recording?.isPlaying() && speed !== 1) {
      // Switching from native 1x to custom speed
      recording.pause()
      startSpeedPlayback()
    }
  }

  function startSpeedPlayback() {
    if (!recording || speedTimer) return
    ended.value = false
    const speed = playbackSpeed.value
    const tickMs = SPEED_TICK_MS
    const positionIncrement = speed * tickMs // ms to advance per tick

    speedTimer = setInterval(() => {
      if (!recording) { stopSpeedPlayback(); return }
      const pos = recording.getPosition()
      const dur = recording.getDuration()
      const nextPos = pos + positionIncrement

      if (nextPos >= dur) {
        // Seek to end and stop
        recording.seek(dur, () => {
          ended.value = true
          paused.value = true
          if (recording) recording.onpause?.()
        })
        stopSpeedPlayback()
        return
      }

      recording.seek(nextPos)
    }, tickMs)

    paused.value = false
    recording.onplay?.()
  }

  function stopSpeedPlayback() {
    if (speedTimer) {
      clearInterval(speedTimer)
      speedTimer = null
    }
  }

  function destroy() {
    stopSpeedPlayback()
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
    playbackSpeed,
    ended,
    load,
    play,
    pause,
    togglePlay,
    seek,
    setSpeed,
    fitDisplay,
    setContainer,
    destroy,
  }
}
