<template>
  <div class="rdp-page">
    <div class="rdp-toolbar">
      <span class="connection-status" :style="{ color: statusColor }">
        {{ statusText }}
      </span>
      <span v-if="hostIp" class="host-info">{{ hostIp }}</span>
      <div class="toolbar-right">
        <el-button size="small" :title="isFullscreen ? '退出全屏' : '全屏'" @click="toggleFullscreen">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
        </el-button>
      </div>
    </div>
    <div ref="displayContainer" class="rdp-display" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { FullScreen, Close } from '@element-plus/icons-vue'
import { useRdpWebSocket } from '../composables/useRdpWebSocket'
import Guacamole from 'guacamole-common-js'

const route = useRoute()
const key = route.query.key as string
const hostIp = route.query.host as string
const wsHost = import.meta.env.VITE_WS_HOST || window.location.host

const displayContainer = ref<HTMLDivElement>()
const isFullscreen = ref(false)

const { status, error, connect } = useRdpWebSocket(key)

const statusColor = computed(() => {
  switch (status.value) {
    case 'connected': return '#67c23a'
    case 'connecting': return '#e6a23c'
    case 'error': return '#f56c6c'
    default: return '#909399'
  }
})

const statusText = computed(() => {
  switch (status.value) {
    case 'connecting': return '连接中...'
    case 'connected': return '已连接'
    case 'disconnected': return '已断开'
    case 'error': return error.value || '错误'
    default: return ''
  }
})

let guacClient: any = null

function onFullscreenChange() {
  isFullscreen.value = !!document.fullscreenElement
}

function toggleFullscreen() {
  if (document.fullscreenElement) {
    document.exitFullscreen()
  } else {
    document.documentElement.requestFullscreen()
  }
}

onMounted(() => {
  if (!key) {
    ElMessage.error('缺少连接密钥')
    return
  }

  document.addEventListener('fullscreenchange', onFullscreenChange)

  const socket = connect(wsHost)

  socket.addEventListener('open', () => {
    // Create a tunnel that adapts our JSON WebSocket to Guacamole protocol format
    const tunnel = new Guacamole.WebSocketTunnel(socket as any)

    // Create Guacamole client with the tunnel
    guacClient = new Guacamole.Client(tunnel)

    // Get display element
    const display = guacClient.getDisplay().getElement()
    displayContainer.value!.appendChild(display)

    // Mouse input
    const mouse = new Guacamole.Mouse(display)
    mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: any) => {
      guacClient.sendMouseState(mouseState)
    }

    // Keyboard input
    const keyboard = new Guacamole.Keyboard(document)
    keyboard.onkeydown = (keysym: number) => {
      guacClient.sendKeyEvent(1, keysym)
    }
    keyboard.onkeyup = (keysym: number) => {
      guacClient.sendKeyEvent(0, keysym)
    }

    // Send initial size
    const width = displayContainer.value?.clientWidth || 1024
    const height = displayContainer.value?.clientHeight || 768
    socket.send(JSON.stringify({ width, height }))

    // Handle resize
    window.addEventListener('resize', () => {
      if (displayContainer.value) {
        socket.send(JSON.stringify({
          width: displayContainer.value.clientWidth,
          height: displayContainer.value.clientHeight,
        }))
      }
    })
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  guacClient?.disconnect()
})
</script>

<style scoped>
.rdp-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1e1e1e;
}

.rdp-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
}

.connection-status {
  font-size: 13px;
}

.host-info {
  font-size: 11px;
  color: #909399;
}

.toolbar-right {
  display: flex;
  gap: 6px;
  margin-left: auto;
  align-items: center;
}

.rdp-display {
  flex: 1;
  overflow: hidden;
}
</style>
