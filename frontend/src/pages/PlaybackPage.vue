<template>
  <div class="playback-page">
    <div class="playback-toolbar">
      <span class="back-link" @click="$router.back()">← 返回审计日志</span>
      <span class="toolbar-sep">|</span>
      <span class="toolbar-title">会话回放</span>
      <span class="toolbar-key">{{ key }}</span>
      <span class="toolbar-protocol">{{ protocol === 'rdp' ? 'Windows (RDP)' : 'Linux (SSH)' }}</span>
    </div>
    <div v-if="loading" class="loading">
      <span>加载录制中...</span>
    </div>
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <el-button @click="$router.back()">返回</el-button>
    </div>
    <div v-else ref="playerContainer" class="player-container" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useAudit } from '../composables/useAudit'

const route = useRoute()
const key = route.query.key as string
const protocol = (route.query.protocol as string) || 'ssh'

const { fetchRecordUrl, fetchGuacRecordUrl } = useAudit()
const playerContainer = ref<HTMLDivElement>()
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  if (!key) {
    error.value = '缺少 key 参数'
    loading.value = false
    return
  }

  try {
    if (protocol === 'rdp') {
      await playGuacRecording()
    } else {
      await playAsciinemaRecording()
    }
  } catch (e: any) {
    error.value = e?.message || '加载录制失败'
    loading.value = false
  }
})

async function playAsciinemaRecording() {
  const AsciinemaPlayer = await import('asciinema-player')
  await import('asciinema-player/dist/bundle/asciinema-player.css')

  const url = await fetchRecordUrl(key)
  loading.value = false
  await nextTick()
  if (playerContainer.value) {
    AsciinemaPlayer.create(url, playerContainer.value, {
      autoPlay: true,
      speed: 1.0,
      idleTimeLimit: 2,
    })
  }
}

async function playGuacRecording() {
  const { default: Guacamole } = await import('guacamole-common-js')

  const url = fetchGuacRecordUrl(key)

  // Fetch the .guac file as a Blob
  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`)
  }
  const blob = await response.blob()

  // Create SessionRecording from the Blob
  const recording = new Guacamole.SessionRecording(blob)

  loading.value = false
  await nextTick()

  if (!playerContainer.value) return

  // Create display element
  const playbackClient = new (Guacamole as any).Client(null)
  const displayElement = playbackClient.getDisplay().getElement()
  displayElement.style.margin = '0 auto'
  displayElement.style.display = 'block'
  playerContainer.value.appendChild(displayElement)

  // Connect recording to the playback client
  recording.connect()

  // Forward recording instructions to the display
  recording.oninstruction = (opcode: string, args: string[]) => {
    playbackClient.getDisplay().eval(opcode, args)
  }

  // Auto-play when connection is established
  recording.onstatechange = (state: number) => {
    if (state === (Guacamole as any).Client.State.OPEN) {
      recording.play()
    }
  }

  // Handle errors
  recording.onerror = (errorMsg: any) => {
    error.value = `RDP录制加载失败: ${errorMsg}`
  }
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
  padding: 24px;
  max-width: 960px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
