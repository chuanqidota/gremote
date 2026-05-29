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
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { FullScreen, Close } from '@element-plus/icons-vue'
import { useFullscreen } from '../composables/useFullscreen'
import { useConnectionStatus } from '../composables/useConnectionStatus'
import Guacamole from 'guacamole-common-js'

const route = useRoute()
const key = route.query.key as string
const hostIp = route.query.host as string

const displayContainer = ref<HTMLDivElement>()

const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')
const error = ref('')

const { statusColor, statusText } = useConnectionStatus(status, error)
const { isFullscreen, toggleFullscreen } = useFullscreen(onFullscreenChange)

let guacClient: any = null
let tunnel: Guacamole.WebSocketTunnel | null = null

function getToolbarHeight(): number {
  const toolbarEl = document.querySelector('.rdp-toolbar') as HTMLElement
  return toolbarEl ? toolbarEl.offsetHeight : 0
}

function updateContainerSize() {
  if (!displayContainer.value) return
  const toolbarH = isFullscreen.value ? 0 : getToolbarHeight()
  displayContainer.value.style.top = toolbarH + 'px'
  displayContainer.value.style.width = window.innerWidth + 'px'
  displayContainer.value.style.height = (window.innerHeight - toolbarH) + 'px'
}

function fitDisplay() {
  if (!guacClient || !displayContainer.value) return
  const display = guacClient.getDisplay()
  const container = displayContainer.value
  const containerW = container.clientWidth
  const containerH = container.clientHeight
  const displayW = display.getWidth()
  const displayH = display.getHeight()
  if (displayW === 0 || displayH === 0 || containerW === 0 || containerH === 0) return
  const scale = Math.min(containerW / displayW, containerH / displayH)
  display.scale(scale)
}

function onFullscreenChange() {
  const toolbarEl = document.querySelector('.rdp-toolbar') as HTMLElement
  if (toolbarEl) {
    toolbarEl.style.display = isFullscreen.value ? 'none' : 'flex'
  }
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      updateContainerSize()
      fitDisplay()
      if (guacClient && status.value === 'connected') {
        guacClient.sendSize(window.innerWidth, window.innerHeight)
      }
      setTimeout(() => {
        updateContainerSize()
        fitDisplay()
      }, 300)
    })
  })
}

onMounted(() => {
  if (!key) {
    ElMessage.error('缺少连接密钥')
    return
  }

  const pageEl = document.querySelector('.rdp-page') as HTMLElement
  if (pageEl) {
    pageEl.style.position = 'relative'
    pageEl.style.height = '100vh'
    pageEl.style.overflow = 'hidden'
  }

  const backendHost = import.meta.env.VITE_API_HOST || 'localhost:8000'
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const toolbarH = getToolbarHeight()
  const wsUrl = `${protocol}//${backendHost}/ws/v1/rdp/${key}?width=${window.innerWidth}&height=${window.innerHeight - toolbarH}`

  tunnel = new Guacamole.WebSocketTunnel(wsUrl)
  guacClient = new Guacamole.Client(tunnel)

  const displayEl = guacClient.getDisplay().getElement()
  displayContainer.value!.appendChild(displayEl)
  displayEl.style.cursor = 'none'

  const container = displayContainer.value!
  container.style.position = 'fixed'
  container.style.left = '0'
  container.style.overflow = 'hidden'
  container.style.zIndex = '1'
  updateContainerSize()

  const guacDisplay = guacClient.getDisplay()

  const mouse = new Guacamole.Mouse(displayEl)
  mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: any) => {
    guacClient.sendMouseState(mouseState, true)
  }

  const keyboard = new Guacamole.Keyboard(document)
  keyboard.onkeydown = (keysym: number) => {
    guacClient.sendKeyEvent(1, keysym)
  }
  keyboard.onkeyup = (keysym: number) => {
    guacClient.sendKeyEvent(0, keysym)
  }

  tunnel.onerror = (errorMsg: any) => {
    console.error('[RDP] Tunnel error:', errorMsg)
    status.value = 'error'
    error.value = (errorMsg && errorMsg.message) || '连接失败'
    ElMessage.error(error.value)
  }

  tunnel.onstatechange = (tunnelState: number) => {
    switch (tunnelState) {
      case Guacamole.Tunnel.State.OPEN:
        status.value = 'connected'
        break
      case Guacamole.Tunnel.State.CLOSED:
        status.value = 'disconnected'
        break
      case Guacamole.Tunnel.State.CONNECTING:
        status.value = 'connecting'
        break
    }
  }

  guacClient.connect('')

  guacDisplay.onresize = () => {
    fitDisplay()
  }

  let synced = false
  guacClient.onsync = () => {
    if (!synced) {
      synced = true
      fitDisplay()
    }
  }

  let resizeTimer: ReturnType<typeof setTimeout> | null = null
  window.addEventListener('resize', () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      if (displayContainer.value && guacClient) {
        updateContainerSize()
        fitDisplay()
        if (!isFullscreen.value && guacClient && status.value === 'connected') {
          const toolbarH = getToolbarHeight()
          guacClient.sendSize(window.innerWidth, window.innerHeight - toolbarH)
        }
      }
    }, 100)
  })
})

onBeforeUnmount(() => {
  tunnel?.disconnect()
  guacClient = null
  tunnel = null
})
</script>

<style scoped>
.rdp-page {
  position: relative;
  height: 100vh;
  background: #1e1e1e;
  overflow: hidden;
}

.rdp-toolbar {
  position: relative;
  z-index: 10;
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
  overflow: hidden;
}
</style>
