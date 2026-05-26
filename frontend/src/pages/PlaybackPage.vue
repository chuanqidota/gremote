<template>
  <div class="playback-page">
    <div class="playback-toolbar">
      <span class="toolbar-title">会话回放</span>
      <span class="toolbar-key">{{ key }}</span>
      <span class="toolbar-protocol">{{ isRdp ? 'Windows (RDP)' : 'Linux (SSH)' }}</span>
      <div class="toolbar-right">
        <el-button v-if="isRdp" size="small" :title="isFullscreen ? '退出全屏' : '全屏'" @click="toggleFullscreen">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
        </el-button>
      </div>
    </div>

    <div v-if="showLoading" class="loading">
      <el-progress :percentage="Math.round(loadProgress * 100)" :stroke-width="8" style="width: 300px" />
      <span>{{ loadLabel }}</span>
    </div>

    <div v-else-if="showError" class="error">
      <p>{{ errorMsg }}</p>
      <div class="error-actions">
        <el-button @click="retry">重试</el-button>
        <el-button @click="$router.back()">返回</el-button>
      </div>
    </div>

    <template v-else>
      <div ref="playerContainer" class="player-container" />
      <div v-if="isSeeking" class="seek-overlay">
        <span>正在跳转... {{ Math.round(seekProgressVal * 100) }}%</span>
      </div>
      <div class="playback-controls">
        <el-button size="small" @click="togglePlay">{{ paused ? '▶ 播放' : '⏸ 暂停' }}</el-button>
        <el-slider :model-value="progress" :max="100" :show-tooltip="true" :format-tooltip="fmtTooltip" style="flex: 1" @change="onSeek" />
        <span class="time-display">{{ fmtTime(currentTime) }} / {{ fmtTime(totalDuration) }}</span>
        <el-dropdown trigger="click" @command="onSetSpeed">
          <el-button size="small" class="speed-btn">{{ playbackSpeed }}x</el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item :command="0.5">0.5x</el-dropdown-item>
              <el-dropdown-item :command="1">1x</el-dropdown-item>
              <el-dropdown-item :command="2">2x</el-dropdown-item>
              <el-dropdown-item :command="4">4x</el-dropdown-item>
              <el-dropdown-item :command="8">8x</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
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

interface AsciinemaEvent { time: number; type: string; data: string }

const route = useRoute()
const key = route.query.key as string
const protocol = (route.query.protocol as string) || 'ssh'
const isRdp = protocol === 'rdp'

const { fetchRecordUrl, fetchGuacRecordUrl } = useAudit()
const guacPlayback = useGuacPlayback()
const playerContainer = ref<HTMLDivElement>()
const isFullscreen = ref(false)

// --- Unified state (RDP uses guacPlayback, SSH uses local refs) ---
const ssh = isRdp ? null : {
  loading: ref(true),
  error: ref(''),
  paused: ref(true),
  progress: ref(0),
  currentTime: ref(0),
  duration: ref(0),
}

const sshSpeed = ref(1)
const playbackSpeed = computed(() => isRdp ? guacPlayback.playbackSpeed.value : sshSpeed.value)

const showLoading = computed(() => isRdp ? guacPlayback.loading.value : ssh!.loading.value)
const showError = computed(() => isRdp ? guacPlayback.error.value : ssh!.error.value)
const errorMsg = computed(() => isRdp ? guacPlayback.error.value : ssh!.error.value)
const paused = computed(() => isRdp ? guacPlayback.paused.value : ssh!.paused.value)
const progress = computed(() => isRdp ? guacPlayback.progress.value : ssh!.progress.value)
const currentTime = computed(() => isRdp ? guacPlayback.currentTime.value : ssh!.currentTime.value)
const totalDuration = computed(() => isRdp ? guacPlayback.duration.value : ssh!.duration.value)
const loadProgress = computed(() => isRdp ? guacPlayback.loadingProgress.value : 0)
const loadLabel = computed(() => isRdp ? guacPlayback.loadingLabel.value : '')
const isSeeking = computed(() => isRdp && guacPlayback.seeking.value)
const seekProgressVal = computed(() => isRdp ? guacPlayback.seekProgress.value : 0)

// --- SSH playback internals ---
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let timerIds: number[] = []
let startTime = 0
let pausedAt = 0
let events: AsciinemaEvent[] = []
let eventIndex = 0

function fmtTime(sec: number): string {
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = Math.floor(sec % 60)
  return h > 0
    ? `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
    : `${m}:${String(s).padStart(2, '0')}`
}

function fmtTooltip(val: number): string {
  return fmtTime((val / 100) * totalDuration.value)
}

function togglePlay() {
  if (isRdp) {
    guacPlayback.togglePlay()
  } else if (ssh!.paused.value) {
    resumeSsh()
  } else {
    pauseSsh()
  }
}

function onSeek(pos: number) {
  if (isRdp) {
    guacPlayback.seek(pos)
  } else {
    seekSsh(pos)
  }
}

function onSetSpeed(speed: number) {
  if (isRdp) {
    guacPlayback.setSpeed(speed)
    return
  }
  const currentPos = ssh!.currentTime.value
  sshSpeed.value = speed
  startTime = performance.now() - currentPos * 1000 / speed
  if (!ssh!.paused.value) {
    clearTimers()
    scheduleSshEvents()
  }
}

function retry() {
  if (isRdp) {
    guacPlayback.error.value = ''
    guacPlayback.loading.value = true
    guacPlayback.loadingProgress.value = 0
    guacPlayback.loadingLabel.value = '正在加载录制文件...'
    guacPlayback.load(fetchGuacRecordUrl(key), () => playerContainer.value).catch((e: any) => {
      guacPlayback.error.value = e?.message || '加载录制失败'
      guacPlayback.loading.value = false
    })
  }
}

// --- Keyboard ---
function onKeydown(e: KeyboardEvent) {
  if (!isRdp) return
  const seekDelta = 5 / totalDuration.value * 100
  switch (e.code) {
    case 'Space': e.preventDefault(); guacPlayback.togglePlay(); break
    case 'ArrowLeft': e.preventDefault(); guacPlayback.seek(Math.max(0, progress.value - seekDelta)); break
    case 'ArrowRight': e.preventDefault(); guacPlayback.seek(Math.min(100, progress.value + seekDelta)); break
    case 'KeyF': e.preventDefault(); toggleFullscreen(); break
  }
}

// --- Fullscreen ---
function toggleFullscreen() {
  document.fullscreenElement ? document.exitFullscreen() : document.documentElement.requestFullscreen()
}

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
  const hide = isFullscreen.value ? 'none' : ''
  const toolbar = document.querySelector('.playback-toolbar') as HTMLElement
  const controls = document.querySelector('.playback-controls') as HTMLElement
  if (toolbar) toolbar.style.display = isFullscreen.value ? 'none' : 'flex'
  if (controls) controls.style.display = hide || 'flex'
  requestAnimationFrame(() => requestAnimationFrame(() => guacPlayback.fitDisplay()))
}

// --- Resize ---
let resizeTimer: ReturnType<typeof setTimeout> | null = null
function onResize() {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => { guacPlayback.fitDisplay(); fitAddon?.fit() }, 100)
}

// --- SSH functions ---
function clearTimers() { timerIds.forEach(clearTimeout); timerIds = [] }

function pauseSsh() {
  ssh!.paused.value = true
  pausedAt = performance.now()
  clearTimers()
}

function resumeSsh() {
  if (!terminal || events.length === 0) return
  ssh!.paused.value = false
  startTime += performance.now() - pausedAt
  scheduleSshEvents()
}

function scheduleSshEvents() {
  clearTimers()
  const now = performance.now()
  while (eventIndex < events.length) {
    const ev = events[eventIndex]
    const elapsed = (now - startTime) * sshSpeed.value
    const delay = ev.time * 1000 - elapsed
    if (delay > 0) {
      timerIds.push(window.setTimeout(() => {
        if (!ssh!.paused.value) {
          terminal?.write(ev.data)
          eventIndex++
          ssh!.currentTime.value = ev.time
          ssh!.progress.value = ssh!.duration.value > 0 ? (ev.time / ssh!.duration.value) * 100 : 0
          scheduleSshEvents()
        }
      }, delay))
      return
    }
    terminal?.write(ev.data)
    eventIndex++
    ssh!.currentTime.value = ev.time
  }
  ssh!.progress.value = 100
}

function seekSsh(pos: number) {
  if (ssh!.duration.value <= 0 || events.length === 0) return
  clearTimers()
  const targetTime = (pos / 100) * ssh!.duration.value
  terminal?.clear()
  terminal?.reset()
  eventIndex = 0
  let acc = 0
  while (eventIndex < events.length && events[eventIndex].time <= targetTime) {
    terminal?.write(events[eventIndex].data)
    acc = events[eventIndex].time
    eventIndex++
  }
  ssh!.currentTime.value = acc
  ssh!.progress.value = pos
  startTime = performance.now() - acc * 1000 / sshSpeed.value
  if (!ssh!.paused.value) scheduleSshEvents()
}

async function initSsh() {
  const resp = await fetch(await fetchRecordUrl(key))
  if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${resp.statusText}`)
  const data = await resp.text()
  if (!data?.trim()) throw new Error('录制数据为空')

  const lines = data.split('\n').filter(l => l.trim())
  const header = JSON.parse(lines[0])
  events = []
  for (let i = 1; i < lines.length; i++) {
    try { const a = JSON.parse(lines[i]); events.push({ time: a[0], type: a[1], data: a[2] || '' }) } catch {}
  }
  if (!events.length) throw new Error('录制数据中没有事件')
  ssh!.duration.value = events[events.length - 1].time
  ssh!.loading.value = false

  await nextTick()
  if (!playerContainer.value) return

  terminal = new Terminal({
    cols: header.cols || 80, rows: header.rows || 24,
    fontSize: 14, fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: { background: '#1e1e1e', foreground: '#d4d4d4', cursor: '#d4d4d4' },
    cursorBlink: false, disableStdin: true, allowProposedApi: true,
  })
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(playerContainer.value)
  fitAddon.fit()
  startTime = performance.now()
  sshSpeed.value = 1
  scheduleSshEvents()
}

// --- Lifecycle ---
onMounted(async () => {
  if (!key) { if (!isRdp) { ssh!.error.value = '缺少 key 参数'; ssh!.loading.value = false }; return }
  try {
    isRdp ? await guacPlayback.load(fetchGuacRecordUrl(key), () => playerContainer.value) : await initSsh()
  } catch (e: any) {
    const msg = e?.message || '加载录制失败'
    isRdp ? (guacPlayback.error.value = msg, guacPlayback.loading.value = false) : (ssh!.error.value = msg, ssh!.loading.value = false)
  }
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('fullscreenchange', onFullscreenChange)
  window.addEventListener('resize', onResize)
})

onBeforeUnmount(() => {
  clearTimers()
  guacPlayback.destroy()
  terminal?.dispose()
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.playback-page { display: flex; flex-direction: column; height: 100vh; background: #1e1e1e }
.playback-toolbar { display: flex; align-items: center; gap: 8px; padding: 6px 14px; background: #2d2d2d; border-bottom: 1px solid #3d3d3d; flex-shrink: 0 }
.toolbar-title { font-size: 12px; color: #d4d4d4 }
.toolbar-key { font-size: 11px; color: #909399 }
.toolbar-protocol { font-size: 11px; color: #67c23a; background: rgba(103,194,58,0.1); padding: 1px 6px; border-radius: 2px }
.toolbar-right { display: flex; gap: 6px; margin-left: auto; align-items: center }
.loading, .error { display: flex; flex-direction: column; align-items: center; justify-content: center; flex: 1; gap: 12px; color: #ccc; font-size: 16px }
.error-actions { display: flex; gap: 8px }
.player-container { flex: 1; min-height: 0; display: flex; align-items: center; justify-content: center; overflow: hidden }
.seek-overlay { position: absolute; top: 50%; left: 50%; transform: translate(-50%,-50%); background: rgba(0,0,0,0.7); color: #fff; padding: 12px 24px; border-radius: 8px; font-size: 14px; z-index: 10; pointer-events: none }
.playback-controls { display: flex; align-items: center; gap: 12px; padding: 8px 16px; background: #2d2d2d; border-top: 1px solid #3d3d3d; flex-shrink: 0 }
.time-display { font-size: 12px; color: #909399; font-variant-numeric: tabular-nums; white-space: nowrap }
.speed-btn { font-size: 12px; min-width: 48px }
</style>
