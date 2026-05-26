<template>
  <div class="playback-page">
    <div class="playback-toolbar">
      <span class="back-link" @click="$router.back()">← 返回审计日志</span>
      <span class="toolbar-sep">|</span>
      <span class="toolbar-title">会话回放</span>
      <span class="toolbar-key">{{ key }}</span>
      <span class="toolbar-protocol">{{ protocol === 'rdp' ? 'Windows (RDP)' : 'Linux (SSH)' }}</span>
    </div>

    <!-- Converting state -->
    <div v-if="converting" class="loading">
      <el-icon class="converting-spinner" :size="48"><Loading /></el-icon>
      <span>正在转换为MP4视频...</span>
      <span class="converting-hint">首次播放需要转换，转换完成后将自动播放</span>
    </div>

    <!-- Loading state -->
    <div v-else-if="loading" class="loading">
      <span>加载录制中...</span>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <el-button @click="$router.back()">返回</el-button>
    </div>

    <!-- MP4 Video player -->
    <template v-else-if="useMP4Player">
      <div class="player-container">
        <video
          ref="videoPlayer"
          :src="mp4Url"
          autoplay
          class="video-player"
          @loadedmetadata="onVideoLoaded"
          @timeupdate="onVideoTimeUpdate"
          @ended="onVideoEnded"
        />
      </div>
      <div class="playback-controls">
        <el-button size="small" @click="toggleVideoPlay">
          {{ videoPaused ? '▶ 播放' : '⏸ 暂停' }}
        </el-button>
        <el-slider
          v-model="videoProgress"
          :max="100"
          :show-tooltip="false"
          style="flex: 1"
          @input="onVideoSeek"
        />
        <span class="time-display">{{ formatTime(videoCurrentTime) }} / {{ formatTime(videoDuration) }}</span>
        <el-select v-model="videoPlaybackSpeed" size="small" style="width: 80px" @change="onVideoSpeedChange">
          <el-option :value="0.5" label="0.5x" />
          <el-option :value="1" label="1x" />
          <el-option :value="2" label="2x" />
          <el-option :value="5" label="5x" />
          <el-option :value="10" label="10x" />
        </el-select>
      </div>
    </template>

    <!-- Fallback: original .guac player -->
    <template v-else>
      <div ref="playerContainer" class="player-container" />
      <div class="playback-controls">
        <el-button size="small" @click="togglePlay">
          {{ paused ? '▶ 播放' : '⏸ 暂停' }}
        </el-button>
        <el-slider
          v-model="progress"
          :max="100"
          :show-tooltip="false"
          style="flex: 1"
          @input="onSeek"
        />
        <span class="time-display">{{ formatTime(currentTime) }} / {{ formatTime(duration) }}</span>
        <el-select v-model="playbackSpeed" size="small" style="width: 80px" @change="onSpeedChange">
          <el-option :value="0.5" label="0.5x" />
          <el-option :value="1" label="1x" />
          <el-option :value="2" label="2x" />
          <el-option :value="5" label="5x" />
          <el-option :value="10" label="10x" />
        </el-select>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { Loading } from '@element-plus/icons-vue'
import { useAudit } from '../composables/useAudit'
import { getConvertStatus, triggerConvert, getRecordFileMP4Url } from '../api'

interface AsciinemaEvent {
  time: number
  type: string
  data: string
}

interface GuacInstruction {
  opcode: string
  args: string[]
  time: number
}

interface StreamAccumulator {
  mimetype: string
  x: number
  y: number
  channelMask: number
  layerIndex: number
  chunks: string[]
}

const route = useRoute()
const key = route.query.key as string
const protocol = (route.query.protocol as string) || 'ssh'

const { fetchRecordUrl, fetchGuacRecordUrl } = useAudit()
const playerContainer = ref<HTMLDivElement>()
const loading = ref(true)
const converting = ref(false)
const error = ref('')
const useMP4Player = ref(false)
const mp4Url = ref('')
const videoPlayer = ref<HTMLVideoElement>()

// MP4 video player state
const videoPaused = ref(false)
const videoProgress = ref(0)
const videoCurrentTime = ref(0)
const videoDuration = ref(0)
const videoPlaybackSpeed = ref(1)

// Original .guac player state
const paused = ref(false)
const progress = ref(0)
const currentTime = ref(0)
const duration = ref(0)
const playbackSpeed = ref(1)

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let timerIds: number[] = []
let startTime = 0
let pausedAt = 0
let events: AsciinemaEvent[] = []
let eventIndex = 0
let headerCols = 80
let headerRows = 24

// Guac playback state
let guacInstructions: GuacInstruction[] = []
let guacEventIndex = 0
let guacDisplay: any = null
let guacLayers: Map<number, any> = new Map()
let guacStreamAccumulators: Map<number, StreamAccumulator> = new Map()
let pollTimer: ReturnType<typeof setTimeout> | null = null

onMounted(async () => {
  if (!key) {
    error.value = '缺少 key 参数'
    loading.value = false
    return
  }

  try {
    if (protocol === 'rdp') {
      await initRDPPlayback()
    } else {
      await playAsciinemaRecording()
    }
  } catch (e: any) {
    error.value = e?.message || '加载录制失败'
    loading.value = false
  }
})

onBeforeUnmount(() => {
  clearAllTimers()
  if (pollTimer) clearTimeout(pollTimer)
  terminal?.dispose()
})

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
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

// ============ MP4 Video Player ============

function onVideoLoaded() {
  if (!videoPlayer.value) return
  videoDuration.value = videoPlayer.value.duration
  loading.value = false
}

function onVideoTimeUpdate() {
  if (!videoPlayer.value) return
  videoCurrentTime.value = videoPlayer.value.currentTime
  videoProgress.value = videoDuration.value > 0
    ? (videoPlayer.value.currentTime / videoDuration.value) * 100
    : 0
}

function onVideoEnded() {
  videoPaused.value = true
}

function toggleVideoPlay() {
  if (!videoPlayer.value) return
  if (videoPaused.value) {
    videoPlayer.value.play()
    videoPaused.value = false
  } else {
    videoPlayer.value.pause()
    videoPaused.value = true
  }
}

function onVideoSeek(pos: number) {
  if (!videoPlayer.value || videoDuration.value <= 0) return
  videoPlayer.value.currentTime = (pos / 100) * videoDuration.value
}

function onVideoSpeedChange() {
  if (!videoPlayer.value) return
  videoPlayer.value.playbackRate = videoPlaybackSpeed.value
}

// ============ RDP Init with MP4 conversion ============

async function initRDPPlayback() {
  // Check if MP4 already exists
  const status = await getConvertStatus(key)
  if (status.converted && status.mp4_url) {
    mp4Url.value = status.mp4_url
    useMP4Player.value = true
    loading.value = false
    return
  }

  // Trigger conversion
  converting.value = true
  loading.value = false
  await triggerConvert(key)

  // Poll for completion
  startPolling()
}

function startPolling() {
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = setTimeout(async () => {
    try {
      const status = await getConvertStatus(key)
      if (status.converted && status.mp4_url) {
        // Conversion complete
        mp4Url.value = status.mp4_url
        useMP4Player.value = true
        converting.value = false
        return
      }
      if (!status.converting) {
        // Conversion failed or stopped, fallback to .guac player
        converting.value = false
        await playGuacRecording()
        return
      }
      // Still converting, poll again
      startPolling()
    } catch {
      // On error, fallback to .guac player
      converting.value = false
      await playGuacRecording()
    }
  }, 3000)
}

// ============ Original .guac Player (fallback) ============

function togglePlay() {
  if (paused.value) {
    resumePlayback()
  } else {
    pausePlayback()
  }
}

function pausePlayback() {
  paused.value = true
  pausedAt = performance.now()
  clearAllTimers()
}

function resumePlayback() {
  if (protocol === 'rdp') {
    if (guacInstructions.length === 0) return
  } else {
    if (!terminal || events.length === 0) return
  }
  paused.value = false
  const pauseDuration = performance.now() - pausedAt
  startTime += pauseDuration
  if (protocol === 'rdp') {
    scheduleRemainingGuacEvents()
  } else {
    scheduleRemainingEvents()
  }
}

function scheduleRemainingEvents() {
  clearAllTimers()
  const now = performance.now()
  while (eventIndex < events.length) {
    const ev = events[eventIndex]
    const delay = (ev.time * 1000 / playbackSpeed.value) - (now - startTime)
    if (delay > 0) {
      const tid = window.setTimeout(() => {
        if (!paused.value) {
          writeEvent(ev)
          eventIndex++
          currentTime.value = ev.time
          progress.value = duration.value > 0 ? (ev.time / duration.value) * 100 : 0
          scheduleRemainingEvents()
        }
      }, delay)
      timerIds.push(tid)
      return
    }
    writeEvent(ev)
    eventIndex++
    currentTime.value = ev.time
  }
  progress.value = 100
}

function scheduleRemainingGuacEvents() {
  clearAllTimers()
  const now = performance.now()
  while (guacEventIndex < guacInstructions.length) {
    const inst = guacInstructions[guacEventIndex]
    const delay = (inst.time * 1000 / playbackSpeed.value) - (now - startTime)
    if (delay > 0) {
      const tid = window.setTimeout(() => {
        if (!paused.value) {
          processGuacInstruction(inst)
          guacEventIndex++
          currentTime.value = inst.time
          progress.value = duration.value > 0 ? (inst.time / duration.value) * 100 : 0
          scheduleRemainingGuacEvents()
        }
      }, delay)
      timerIds.push(tid)
      return
    }
    processGuacInstruction(inst)
    guacEventIndex++
    currentTime.value = inst.time
  }
  progress.value = 100
}

function writeEvent(ev: AsciinemaEvent) {
  if (!terminal) return
  if (ev.type === 'o') {
    terminal.write(ev.data)
  } else if (ev.type === 'i') {
    // input events are not shown during playback
  }
}

function onSeek(pos: number) {
  if (protocol === 'rdp') {
    onGuacSeek(pos)
  } else {
    onTerminalSeek(pos)
  }
}

function onTerminalSeek(pos: number) {
  if (duration.value <= 0 || events.length === 0) return
  clearAllTimers()
  const targetTime = (pos / 100) * duration.value
  terminal?.clear()
  terminal?.reset()
  eventIndex = 0
  let accumulated = 0
  while (eventIndex < events.length && events[eventIndex].time <= targetTime) {
    writeEvent(events[eventIndex])
    accumulated = events[eventIndex].time
    eventIndex++
  }
  currentTime.value = accumulated
  progress.value = pos
  startTime = performance.now() - (accumulated * 1000 / playbackSpeed.value)
  if (!paused.value) {
    scheduleRemainingEvents()
  }
}

function onGuacSeek(pos: number) {
  if (duration.value <= 0 || guacInstructions.length === 0) return
  clearAllTimers()
  const targetTime = (pos / 100) * duration.value

  // Reset display
  if (guacDisplay) {
    const defaultLayer = guacDisplay.getDefaultLayer()
    const canvas = defaultLayer.getCanvas()
    const ctx = canvas.getContext('2d')
    if (ctx) {
      ctx.clearRect(0, 0, canvas.width, canvas.height)
    }
  }
  guacStreamAccumulators.clear()
  guacLayers.clear()

  // Replay from start up to target time
  guacEventIndex = 0
  let accumulated = 0
  while (guacEventIndex < guacInstructions.length && guacInstructions[guacEventIndex].time <= targetTime) {
    processGuacInstruction(guacInstructions[guacEventIndex])
    accumulated = guacInstructions[guacEventIndex].time
    guacEventIndex++
  }
  currentTime.value = accumulated
  progress.value = pos
  startTime = performance.now() - (accumulated * 1000 / playbackSpeed.value)
  if (!paused.value) {
    scheduleRemainingGuacEvents()
  }
}

function onSpeedChange() {
  if (protocol === 'rdp') {
    if (!paused.value && guacInstructions.length > 0) {
      startTime = performance.now() - (currentTime.value * 1000 / playbackSpeed.value)
      scheduleRemainingGuacEvents()
    }
  } else {
    if (!paused.value && events.length > 0) {
      startTime = performance.now() - (currentTime.value * 1000 / playbackSpeed.value)
      scheduleRemainingEvents()
    }
  }
}

function getGuacLayer(index: number, display: any): any {
  if (guacLayers.has(index)) return guacLayers.get(index)
  let layer: any
  if (index === 0) {
    layer = display.getDefaultLayer()
  } else if (index > 0) {
    layer = display.createLayer()
  } else {
    layer = display.createBuffer()
  }
  guacLayers.set(index, layer)
  return layer
}

function processGuacInstruction(inst: GuacInstruction) {
  if (!guacDisplay) return
  const args = inst.args
  const display = guacDisplay

  switch (inst.opcode) {
    case 'size': {
      const layerIndex = parseInt(args[0])
      const w = parseInt(args[1])
      const h = parseInt(args[2])
      const layer = getGuacLayer(layerIndex, display)
      layer.resize(w, h)
      if (layerIndex === 0) {
        const displayEl = display.getElement()
        const innerDisplay = displayEl.firstChild as HTMLElement
        if (innerDisplay) {
          innerDisplay.style.width = w + 'px'
          innerDisplay.style.height = h + 'px'
        }
        displayEl.style.width = w + 'px'
        displayEl.style.height = h + 'px'
      }
      break
    }
    case 'img': {
      const streamIndex = parseInt(args[0])
      const channelMask = parseInt(args[1])
      const layerIndex = parseInt(args[2])
      const mimetype = args[3]
      const x = parseInt(args[4])
      const y = parseInt(args[5])
      guacStreamAccumulators.set(streamIndex, {
        mimetype, x, y, channelMask, layerIndex, chunks: [],
      })
      break
    }
    case 'blob': {
      const streamIndex = parseInt(args[0])
      const data = args[1]
      const acc = guacStreamAccumulators.get(streamIndex)
      if (acc) acc.chunks.push(data)
      break
    }
    case 'end': {
      const streamIndex = parseInt(args[0])
      const acc = guacStreamAccumulators.get(streamIndex)
      if (acc && acc.chunks.length > 0) {
        drawStreamImage(acc, display)
      }
      guacStreamAccumulators.delete(streamIndex)
      break
    }
    case 'jpeg': {
      const channelMask = parseInt(args[0])
      const layerIndex = parseInt(args[1])
      const x = parseInt(args[2])
      const y = parseInt(args[3])
      const data = args[4]
      const layer = getGuacLayer(layerIndex, display)
      const url = 'data:image/jpeg;base64,' + data
      const img = new Image()
      img.onload = () => layer.drawImage(x, y, img)
      img.src = url
      break
    }
    case 'png': {
      const channelMask = parseInt(args[0])
      const layerIndex = parseInt(args[1])
      const x = parseInt(args[2])
      const y = parseInt(args[3])
      const data = args[4]
      const layer = getGuacLayer(layerIndex, display)
      const url = 'data:image/png;base64,' + data
      const img = new Image()
      img.onload = () => layer.drawImage(x, y, img)
      img.src = url
      break
    }
    case 'cursor': {
      const hotspotX = parseInt(args[0])
      const hotspotY = parseInt(args[1])
      const srcLayerIndex = parseInt(args[2])
      const srcX = parseInt(args[3])
      const srcY = parseInt(args[4])
      const srcWidth = parseInt(args[5])
      const srcHeight = parseInt(args[6])
      const srcLayer = getGuacLayer(srcLayerIndex, display)
      display.setCursor(hotspotX, hotspotY, srcLayer, srcX, srcY, srcWidth, srcHeight)
      break
    }
    case 'mouse': {
      const x = parseInt(args[0])
      const y = parseInt(args[1])
      display.showCursor(true)
      display.moveCursor(x, y)
      break
    }
    case 'cfill': {
      const channelMask = parseInt(args[0])
      const layerIndex = parseInt(args[1])
      const r = parseInt(args[2])
      const g = parseInt(args[3])
      const b = parseInt(args[4])
      const a = parseInt(args[5])
      const layer = getGuacLayer(layerIndex, display)
      layer.setChannelMask(channelMask)
      layer.fillColor(r, g, b, a)
      break
    }
    case 'cstroke': {
      const channelMask = parseInt(args[0])
      const layerIndex = parseInt(args[1])
      const cap = ['butt', 'round', 'square'][parseInt(args[2])]
      const join = ['bevel', 'miter', 'round'][parseInt(args[3])]
      const thickness = parseInt(args[4])
      const r = parseInt(args[5])
      const g = parseInt(args[6])
      const b = parseInt(args[7])
      const a = parseInt(args[8])
      const layer = getGuacLayer(layerIndex, display)
      layer.setChannelMask(channelMask)
      layer.strokeColor(cap, join, thickness, r, g, b, a)
      break
    }
    case 'setChannelMask': {
      const layerIndex = parseInt(args[0])
      const mask = parseInt(args[1])
      const layer = getGuacLayer(layerIndex, display)
      layer.setChannelMask(mask)
      break
    }
    case 'identity': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.setTransform(1, 0, 0, 1, 0, 0)
      break
    }
    case 'setTransform': {
      const layerIndex = parseInt(args[0])
      const a = parseFloat(args[1])
      const b = parseFloat(args[2])
      const c = parseFloat(args[3])
      const d = parseFloat(args[4])
      const e = parseFloat(args[5])
      const f = parseFloat(args[6])
      const layer = getGuacLayer(layerIndex, display)
      layer.setTransform(a, b, c, d, e, f)
      break
    }
    case 'transform': {
      const layerIndex = parseInt(args[0])
      const a = parseFloat(args[1])
      const b = parseFloat(args[2])
      const c = parseFloat(args[3])
      const d = parseFloat(args[4])
      const e = parseFloat(args[5])
      const f = parseFloat(args[6])
      const layer = getGuacLayer(layerIndex, display)
      layer.transform(a, b, c, d, e, f)
      break
    }
    case 'start': {
      const layerIndex = parseInt(args[0])
      const x = parseInt(args[1])
      const y = parseInt(args[2])
      const layer = getGuacLayer(layerIndex, display)
      layer.moveTo(x, y)
      break
    }
    case 'line': {
      const layerIndex = parseInt(args[0])
      const x = parseInt(args[1])
      const y = parseInt(args[2])
      const layer = getGuacLayer(layerIndex, display)
      layer.lineTo(x, y)
      break
    }
    case 'close': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.close()
      break
    }
    case 'rect': {
      const layerIndex = parseInt(args[0])
      const x = parseInt(args[1])
      const y = parseInt(args[2])
      const w = parseInt(args[3])
      const h = parseInt(args[4])
      const layer = getGuacLayer(layerIndex, display)
      layer.rect(x, y, w, h)
      break
    }
    case 'clip': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.clip()
      break
    }
    case 'push': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.push()
      break
    }
    case 'pop': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.pop()
      break
    }
    case 'reset': {
      const layerIndex = parseInt(args[0])
      const layer = getGuacLayer(layerIndex, display)
      layer.reset()
      break
    }
    case 'move': {
      const layerIndex = parseInt(args[0])
      const parentIndex = parseInt(args[1])
      const x = parseInt(args[2])
      const y = parseInt(args[3])
      const z = parseInt(args[4])
      if (layerIndex > 0 && parentIndex >= 0) {
        const layer = getGuacLayer(layerIndex, display)
        const parent = getGuacLayer(parentIndex, display)
        layer.move(parent, x, y, z)
      }
      break
    }
    case 'dispose': {
      const layerIndex = parseInt(args[0])
      if (layerIndex > 0) {
        const layer = getGuacLayer(layerIndex, display)
        layer.dispose()
        guacLayers.delete(layerIndex)
      }
      break
    }
    case 'shade': {
      const layerIndex = parseInt(args[0])
      const alpha = parseInt(args[1])
      if (layerIndex >= 0) {
        const layer = getGuacLayer(layerIndex, display)
        layer.shade(alpha)
      }
      break
    }
    case 'distort': {
      const layerIndex = parseInt(args[0])
      const a = parseFloat(args[1])
      const b = parseFloat(args[2])
      const c = parseFloat(args[3])
      const d = parseFloat(args[4])
      const e = parseFloat(args[5])
      const f = parseFloat(args[6])
      if (layerIndex >= 0) {
        const layer = getGuacLayer(layerIndex, display)
        layer.distort(a, b, c, d, e, f)
      }
      break
    }
    case 'copy': {
      const srcL = getGuacLayer(parseInt(args[0]), display)
      const srcX = parseInt(args[1])
      const srcY = parseInt(args[2])
      const srcW = parseInt(args[3])
      const srcH = parseInt(args[4])
      const channelMask = parseInt(args[5])
      const dstL = getGuacLayer(parseInt(args[6]), display)
      const dstX = parseInt(args[7])
      const dstY = parseInt(args[8])
      dstL.setChannelMask(channelMask)
      dstL.copy(srcL, srcX, srcY, srcW, srcH, dstX, dstY)
      break
    }
    case 'put': {
      const srcL = getGuacLayer(parseInt(args[0]), display)
      const srcX = parseInt(args[1])
      const srcY = parseInt(args[2])
      const srcW = parseInt(args[3])
      const srcH = parseInt(args[4])
      const dstL = getGuacLayer(parseInt(args[5]), display)
      const dstX = parseInt(args[6])
      const dstY = parseInt(args[7])
      dstL.put(srcL, srcX, srcY, srcW, srcH, dstX, dstY)
      break
    }
    case 'transfer': {
      const srcL = getGuacLayer(parseInt(args[0]), display)
      const srcX = parseInt(args[1])
      const srcY = parseInt(args[2])
      const srcW = parseInt(args[3])
      const srcH = parseInt(args[4])
      const funcIndex = parseInt(args[5])
      const dstL = getGuacLayer(parseInt(args[6]), display)
      const dstX = parseInt(args[7])
      const dstY = parseInt(args[8])
      // Guacamole default transfer functions
      const transferFunctions: Record<number, (src: any, dst: any) => void> = {
        0: (src, dst) => { dst.red = dst.green = dst.blue = 0 },
        15: (src, dst) => { dst.red = dst.green = dst.blue = 255 },
        3: (src, dst) => { dst.red = src.red; dst.green = src.green; dst.blue = src.blue },
        12: (src, dst) => { dst.red = src.red; dst.green = src.green; dst.blue = src.blue; dst.alpha = src.alpha },
      }
      const fn = transferFunctions[funcIndex]
      if (fn) {
        dstL.transfer(srcL, srcX, srcY, srcW, srcH, dstX, dstY, fn)
      }
      break
    }
    case 'set': {
      const layerIndex = parseInt(args[0])
      const name = args[1]
      const value = args[2]
      const layer = getGuacLayer(layerIndex, display)
      if (name === 'miter-limit') {
        layer.setMiterLimit?.(parseFloat(value))
      }
      break
    }
    case 'sync': {
      // No-op for playback — sync is only needed for live connections
      break
    }
    case 'body':
    case 'file':
    case 'clipboard':
    case 'pipe':
    case 'audio':
    case 'video':
    case 'msg':
    case 'name':
    case 'error':
    case 'disconnect':
    case 'nop':
    case 'nest':
    case 'required':
    case 'argv':
    case 'filesystem':
    case 'key':
    case 'optimize':
      // Ignored for playback
      break
    default:
      // Unknown instruction — ignore
      break
  }
}

function drawStreamImage(acc: StreamAccumulator, display: any) {
  const base64 = acc.chunks.join('')
  try {
    const byteString = atob(base64)
    const ab = new ArrayBuffer(byteString.length)
    const ia = new Uint8Array(ab)
    for (let i = 0; i < byteString.length; i++) {
      ia[i] = byteString.charCodeAt(i)
    }
    const blob = new Blob([ab], { type: acc.mimetype })
    const url = URL.createObjectURL(blob)
    const img = new Image()
    img.onload = () => {
      const layer = getGuacLayer(acc.layerIndex, display)
      layer.drawImage(acc.x, acc.y, img)
      URL.revokeObjectURL(url)
    }
    img.onerror = () => {
      console.error('[Playback] Image load error for', acc.mimetype, 'blob size:', blob.size)
      URL.revokeObjectURL(url)
    }
    img.src = url
  } catch (e) {
    console.error('[Playback] Failed to decode image stream:', e)
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
  duration.value = events[events.length - 1].time

  loading.value = false
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

async function playGuacRecording() {
  const { default: Guacamole } = await import('guacamole-common-js')

  const url = fetchGuacRecordUrl(key)
  const response = await fetch(url)
  if (!response.ok) {
    let msg = `HTTP ${response.status}: ${response.statusText}`
    try {
      const errBody = await response.text()
      if (errBody) msg += ` - ${errBody}`
    } catch {}
    throw new Error(msg)
  }

  const arrayBuffer = await response.arrayBuffer()
  if (arrayBuffer.byteLength === 0) {
    throw new Error('录制文件为空')
  }

  const decoder = new TextDecoder('latin1')
  const recordingText = decoder.decode(arrayBuffer)

  const parser = new Guacamole.Parser()
  const rawInstructions: Array<{ opcode: string; args: string[] }> = []
  parser.oninstruction = (opcode: string, args: string[]) => {
    rawInstructions.push({ opcode, args: [...args] })
  }
  parser.receive(recordingText)

  if (rawInstructions.length === 0) {
    throw new Error('录制数据中没有有效指令')
  }

  // Assign timestamps from sync instructions for playback timing
  let currentTimeAccum = 0
  guacInstructions = rawInstructions.map(inst => {
    if (inst.opcode === 'sync') {
      currentTimeAccum += 0.1 // 100ms between syncs as default frame interval
    }
    return { ...inst, time: currentTimeAccum }
  })

  duration.value = currentTimeAccum > 0 ? currentTimeAccum : rawInstructions.length * 0.01

  loading.value = false
  await nextTick()

  if (!playerContainer.value) return

  // Create Display directly — bypass Client to avoid async task system
  guacDisplay = new Guacamole.Display()
  const displayElement = guacDisplay.getElement()
  displayElement.style.margin = '0 auto'
  displayElement.style.display = 'block'
  playerContainer.value.appendChild(displayElement)

  // Start playback
  startTime = performance.now()
  guacEventIndex = 0
  scheduleRemainingGuacEvents()
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

.back-link {
  font-size: 12px;
  color: #ccc;
  cursor: pointer;
}

.toolbar-sep {
  color: #555;
  font-size: 12px;
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

.player-container {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
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

.video-player {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.converting-spinner {
  animation: spin 1s linear infinite;
  color: #409eff;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.converting-hint {
  font-size: 12px;
  color: #909399;
}
</style>
