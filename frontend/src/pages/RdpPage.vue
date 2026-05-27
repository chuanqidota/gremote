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
import Guacamole from 'guacamole-common-js'

const route = useRoute()
const key = route.query.key as string
const hostIp = route.query.host as string

const displayContainer = ref<HTMLDivElement>()
const isFullscreen = ref(false)

const status = ref<'connecting' | 'connected' | 'disconnected' | 'error'>('connecting')
const error = ref('')

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

// Scale Guacamole display to fill the container using CSS transform
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
  isFullscreen.value = !!document.fullscreenElement
  // Hide toolbar in fullscreen so remote desktop gets full screen area
  const toolbarEl = document.querySelector('.rdp-toolbar') as HTMLElement
  if (toolbarEl) {
    toolbarEl.style.display = isFullscreen.value ? 'none' : 'flex'
  }
  // Use requestAnimationFrame to wait for browser to finish fullscreen animation
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      updateContainerSize()
      fitDisplay()
      // Resize RDP session to match the new viewport so the display fills the screen.
      if (guacClient && status.value === 'connected') {
        guacClient.sendSize(window.innerWidth, window.innerHeight)
      }
      // Retry fitDisplay after sendSize takes effect (guacd is async)
      setTimeout(() => {
        updateContainerSize()
        fitDisplay()
      }, 300)
    })
  })
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

  // Force page layout via JavaScript (scoped CSS may not apply to dynamic elements)
  const pageEl = document.querySelector('.rdp-page') as HTMLElement
  if (pageEl) {
    pageEl.style.position = 'relative'
    pageEl.style.height = '100vh'
    pageEl.style.overflow = 'hidden'
  }

  // Build WebSocket URL with viewport dimensions (excluding toolbar)
  const backendHost = import.meta.env.VITE_API_HOST || 'localhost:8000'
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const toolbarH = getToolbarHeight()
  const wsUrl = `${protocol}//${backendHost}/ws/v1/rdp/${key}?width=${window.innerWidth}&height=${window.innerHeight - toolbarH}`

  // Create Guacamole tunnel, client, and display
  tunnel = new Guacamole.WebSocketTunnel(wsUrl)
  guacClient = new Guacamole.Client(tunnel)

  // Attach display element to DOM
  const displayEl = guacClient.getDisplay().getElement()
  displayContainer.value!.appendChild(displayEl)

  // Hide system cursor (guacamole renders its own remote cursor)
  displayEl.style.cursor = 'none'

  // Force container to fill viewport below toolbar (CSS scoped rules may not apply)
  const container = displayContainer.value!
  container.style.position = 'fixed'
  container.style.left = '0'
  container.style.overflow = 'hidden'
  container.style.zIndex = '1'
  updateContainerSize()

  const guacDisplay = guacClient.getDisplay()

  // Mouse input — applyDisplayScale=true corrects coordinates for CSS transform scaling
  const mouse = new Guacamole.Mouse(displayEl)
  mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: any) => {
    guacClient.sendMouseState(mouseState, true)
  }

  // Keyboard input
  const keyboard = new Guacamole.Keyboard(document)
  keyboard.onkeydown = (keysym: number) => {
    guacClient.sendKeyEvent(1, keysym)
  }
  keyboard.onkeyup = (keysym: number) => {
    guacClient.sendKeyEvent(0, keysym)
  }

  // Tunnel event handlers (set before connect so Client.connect()
  // can wrap them without interference from our logging)
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

  // Connect — Client.connect() sets up its own oninstruction handler
  guacClient.connect('')

  // Re-fit display when guacd resizes it
  guacDisplay.onresize = () => {
    fitDisplay()
  }

  // 首次同步时缩放显示以适配容器
  let synced = false
  guacClient.onsync = () => {
    if (!synced) {
      synced = true
      fitDisplay()
    }
  }

  // Handle resize — update container size, fit display, and remote resolution
  // Debounce the entire handler to prevent resize spirals (fitDisplay → scale change → resize → ...)
  let resizeTimer: ReturnType<typeof setTimeout> | null = null
  window.addEventListener('resize', () => {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(() => {
      if (displayContainer.value && guacClient) {
        updateContainerSize()
        fitDisplay()
        // Only sendSize in non-fullscreen mode; fullscreen uses CSS scaling
        if (!isFullscreen.value && guacClient && status.value === 'connected') {
          const toolbarH = getToolbarHeight()
          guacClient.sendSize(window.innerWidth, window.innerHeight - toolbarH)
        }
      }
    }, 100)
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
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
