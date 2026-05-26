<template>
  <div class="playback-page">
    <div class="playback-toolbar">
      <span class="toolbar-title">会话回放</span>
      <span class="toolbar-key">{{ key }}</span>
      <span class="toolbar-protocol">{{ protocol === 'rdp' ? 'Windows (RDP)' : 'Linux (SSH)' }}</span>
      <div class="toolbar-right">
        <el-button v-if="protocol === 'rdp'" size="small" :title="isFullscreen ? '退出全屏' : '全屏'" @click="toggleFullscreen">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="isRdpLoading" class="loading">
      <el-progress
        :percentage="Math.round(guacPlayback.loadingProgress.value * 100)"
        :stroke-width="8"
        style="width: 300px"
      />
      <span>{{ guacPlayback.loadingLabel.value }}</span>
    </div>

    <!-- Error state -->
    <div v-else-if="isRdpError" class="error">
      <p>{{ guacPlayback.error.value }}</p>
      <div class="error-actions">
        <el-button @click="handleRetry">重试</el-button>
        <el-button @click="$router.back()">返回</el-button>
      </div>
    </div>

    <!-- Player (SSH or RDP) -->
    <template v-else>
      <div ref="playerContainer" class="player-container" />
      <div v-if="isRdp && guacPlayback.seeking.value" class="seek-overlay">
        <span>正在跳转... {{ Math.round(guacPlayback.seekProgress.value * 100) }}%</span>
      </div>
      <div class="playback-controls">
        <el-button size="small" @click="handleTogglePlay">
          {{ isRdp ? !guacPlayback.paused.value : !sshPaused ? '⏸ 暂停' : '▶ 播放' }}
        </el-button>
        <el-slider
          :model-value="isRdp ? guacPlayback.progress.value : sshProgress"
          :max="100"
          :show-tooltip="true"
          :format-tooltip="formatTooltip"
          style="flex: 1"
          @change="onSeek"
        />
        <span class="time-display">{{ formatTime(isRdp ? guacPlayback.currentTime.value : sshCurrentTime) }} / {{ formatTime(isRdp ? guacPlayback.duration.value : sshDuration) }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { FullScreen, Close } from '@element-plus/icons-vue'
import { useAudit } from '../composables/useAudit'
import { useGuacPlayback } from '../composables/useGuacPlayback'

interface AsciinemaEvent {
  time: number
  type: string
  data: string
}

const route = useRoute()
const key = route.query.key as string
const protocol = (route.query.protocol as string) || 'ssh'
const isRdp = protocol === 'rdp'

const { fetchRecordUrl, fetchGuacRecordUrl } = useAudit()
const guacPlayback = useGuacPlayback()

const playerContainer = ref<HTMLDivElement>()
const isFullscreen = ref(false)

// SSH-specific state (only used for SSH playback)
const sshLoading = ref(true)
const sshError = ref('')
const sshPaused = ref(true)
const sshProgress = ref(0)
const sshCurrentTime = ref(0)
const sshDuration = ref(0)

// Derived state for template
const isRdpLoading = computed(() => isRdp && guacPlayback.loading.value)
const isRdpError = computed(() => isRdp && guacPlayback.error.value)

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let timerIds: number[] = []
let startTime = 0
let pausedAt = 0
let events: AsciinemaEvent[] = []
let eventIndex = 0
let headerCols = 80
let headerRows = 24

onMounted(async () => {
  if (!key) {
    sshError.value = '缺少 key 参数'
    sshLoading.value = false
    return
  }

  try {
    if (isRdp) {
      await guacPlayback.load(fetchGuacRecordUrl(key), () => playerContainer.value)
    } else {
      await playAsciinemaRecording()
    }
  } catch (e: any) {
    if (isRdp) {
      guacPlayback.error.value = e?.message || '加载录制失败'
      guacPlayback.loading.value = false
    } else {
      sshError.value = e?.message || '加载录制失败'
      sshLoading.value = false
    }
  }

  document.addEventListener('keydown', onKeydown)
  document.addEventListener('fullscreenchange', onFullscreenChange)
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  clearAllTimers()
  guacPlayback.destroy()
  terminal?.dispose()
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('resize', onResize)
})

function onKeydown(e: KeyboardEvent) {
  if (e.code === 'Space' && isRdp) {
    e.preventDefault()
    guacPlayback.togglePlay()
  }
}

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
  const toolbarEl = document.querySelector('.playback-toolbar') as HTMLElement
  if (toolbarEl) {
    toolbarEl.style.display = isFullscreen.value ? 'none' : 'flex'
  }
  const controlsEl = document.querySelector('.playback-controls') as HTMLElement
  if (controlsEl) {
    controlsEl.style.display = isFullscreen.value ? 'none' : 'flex'
  }
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      guacPlayback.fitDisplay()
    })
  })
}

let resizeTimer: ReturnType<typeof setTimeout> | null = null
function onResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => {
    guacPlayback.fitDisplay()
    fitAddon?.fit()
  }, 100)
}

function toggleFullscreen() {
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    document.documentElement.requestFullscreen()
  }
}

function clearAllTimers() {
  timerIds.forEach(id => clearTimeout(id))
  timerIds = []
}

function parseAsciinemaData(raw: string): { header: any; events: AsciinemaEvent[] } {
  const lines = raw.split('\n').filter(l => l.trim())
  const header = JSON.parse(lines[0])
  const parsed: AsciinemaEvent[] = []
  for (let i = 1; i < lines.length; i++) {
    try {
      const arr = JSON.parse(lines[i])
      parsed.push({ time: arr[0], type: arr[1], data: arr[2] || '' })
    } catch {
      // skip malformed lines
    }
  }
  return { header, events: parsed }
}

function formatTime(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  if (h > 0) {
    return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
  }
  return `${m}:${s.toString().padStart(2, '0')}`
}

function formatTooltip(val: number): string {
  const dur: number = isRdp ? guacPlayback.duration.value : sshDuration.value
  const seconds = (val / 100) * dur
  return formatTime(seconds)
}

// ============ RDP ============

function handleTogglePlay() {
  if (isRdp) {
    guacPlayback.togglePlay()
  } else {
    if (sshPaused.value) {
      resumePlayback()
    } else {
      pausePlayback()
    }
  }
}

function onSeek(pos: number) {
  if (isRdp) {
    guacPlayback.seek(pos as number)
  } else {
    onTerminalSeek(pos as number)
  }
}

function handleRetry() {
  guacPlayback.error.value = ''
  guacPlayback.loading.value = true
  guacPlayback.loadingProgress.value = 0
  guacPlayback.loadingLabel.value = '正在加载录制文件...'
  guacPlayback.load(fetchGuacRecordUrl(key), () => playerContainer.value).catch((e: any) => {
    guacPlayback.error.value = e?.message || '加载录制失败'
    guacPlayback.loading.value = false
  })
}

// ============ SSH: Terminal-based asciinema playback ============

function pausePlayback() {
  sshPaused.value = true
  pausedAt = performance.now()
  clearAllTimers()
}

function resumePlayback() {
  if (!terminal || events.length === 0) return
  sshPaused.value = false
  const pauseDuration = performance.now() - pausedAt
  startTime += pauseDuration
  scheduleRemainingEvents()
}

function scheduleRemainingEvents() {
  clearAllTimers()
  const now = performance.now()
  while (eventIndex < events.length) {
    const ev = events[eventIndex]
    const delay = (ev.time * 1000) - (now - startTime)
    if (delay > 0) {
      const tid = window.setTimeout(() => {
        if (!sshPaused.value) {
          writeEvent(ev)
          eventIndex++
          sshCurrentTime.value = ev.time
          sshProgress.value = sshDuration.value > 0 ? (ev.time / sshDuration.value) * 100 : 0
          scheduleRemainingEvents()
        }
      }, delay)
      timerIds.push(tid)
      return
    }
    writeEvent(ev)
    eventIndex++
    sshCurrentTime.value = ev.time
  }
  sshProgress.value = 100
}

function writeEvent(ev: AsciinemaEvent) {
  if (!terminal) return
  if (ev.type === 'o') {
    terminal.write(ev.data)
  }
}

function onTerminalSeek(pos: number) {
  if (sshDuration.value <= 0 || events.length === 0) return
  clearAllTimers()
  const targetTime = (pos / 100) * sshDuration.value
  terminal?.clear()
  terminal?.reset()
  eventIndex = 0
  let accumulated = 0
  while (eventIndex < events.length && events[eventIndex].time <= targetTime) {
    writeEvent(events[eventIndex])
    accumulated = events[eventIndex].time
    eventIndex++
  }
  sshCurrentTime.value = accumulated
  sshProgress.value = pos
  startTime = performance.now() - (accumulated * 1000)
  if (!sshPaused.value) {
    scheduleRemainingEvents()
  }
}

async function playAsciinemaRecording() {
  const recordUrl = await fetchRecordUrl(key)
  const resp = await fetch(recordUrl)
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
  }
  const data = await resp.text()
  if (!data || data.trim().length === 0) {
    throw new Error('录制数据为空')
  }

  const parsed = parseAsciinemaData(data)
  headerCols = parsed.header.cols || 80
  headerRows = parsed.header.rows || 24
  events = parsed.events
  if (events.length === 0) {
    throw new Error('录制数据中没有事件')
  }
  sshDuration.value = events[events.length - 1].time

  sshLoading.value = false
  await nextTick()
  if (!playerContainer.value) return

  terminal = new Terminal({
    cols: headerCols,
    rows: headerRows,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
    },
    cursorBlink: false,
    disableStdin: true,
    allowProposedApi: true,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(playerContainer.value)
  fitAddon.fit()

  startTime = performance.now()
  eventIndex = 0
  scheduleRemainingEvents()
}
</script>

<style scoped>
.playback-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1e1e1e;
}

.playback-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
  flex-shrink: 0;
}

.toolbar-title {
  font-size: 12px;
  color: #d4d4d4;
}

.toolbar-key {
  font-size: 11px;
  color: #909399;
}

.toolbar-protocol {
  font-size: 11px;
  color: #67c23a;
  background: rgba(103, 194, 58, 0.1);
  padding: 1px 6px;
  border-radius: 2px;
}

.toolbar-right {
  display: flex;
  gap: 6px;
  margin-left: auto;
  align-items: center;
}

.loading,
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 12px;
  color: #ccc;
  font-size: 16px;
}

.error-actions {
  display: flex;
  gap: 8px;
}

.player-container {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.seek-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background: rgba(0, 0, 0, 0.7);
  color: #fff;
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 14px;
  z-index: 10;
  pointer-events: none;
}

.playback-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: #2d2d2d;
  border-top: 1px solid #3d3d3d;
  flex-shrink: 0;
}

.time-display {
  font-size: 12px;
  color: #909399;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
</style>
