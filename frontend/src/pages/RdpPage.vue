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
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
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
let syncCount = 0
let imgDecoded = 0
let imgDrawn = 0
const instrCounts: Record<string, number> = {}

// Diagnostic: monkey-patch createImageBitmap to track image decoding + fallback on failure
let imgFailed = 0
const _origCreateImageBitmap = window.createImageBitmap
if (_origCreateImageBitmap) {
	window.createImageBitmap = function(blob: any, opts?: any) {
		imgDecoded++
		const tag = imgDecoded
		console.log(`[RDP] createImageBitmap #${tag}: blob size=${blob?.size}, type=${blob?.type}`)
		const promise = (_origCreateImageBitmap as any).call(window, blob, opts) as Promise<ImageBitmap>
		promise.then(
			(bitmap: ImageBitmap) => {
				console.log(`[RDP] createImageBitmap #${tag}: SUCCESS ${bitmap.width}x${bitmap.height}`)
			},
			(err: any) => {
				imgFailed++
				console.error(`[RDP] createImageBitmap #${tag}: FAILED`, err?.message || err,
					`blob size=${blob?.size} type=${blob?.type}`)
			}
		)
		// Chain .catch() fallback: if decode fails, create 1x1 transparent bitmap
		// so task.unblock() still fires and subsequent frames aren't blocked
		return promise.catch(async (err: any) => {
			imgFailed++
			console.error(`[RDP] createImageBitmap #${tag}: FALLBACK`, err?.message || err)
			const c = document.createElement('canvas')
			c.width = 1; c.height = 1
			return (_origCreateImageBitmap as any).call(window, c) as Promise<ImageBitmap>
		})
	}
}

// Diagnostic: intercept Canvas drawImage to track actual pixel rendering
const _origDrawImage = CanvasRenderingContext2D.prototype.drawImage
let _drawLogCount = 0
CanvasRenderingContext2D.prototype.drawImage = function(...args: any[]) {
	if (_drawLogCount < 5) {
		_drawLogCount++
		const img = args[0] as any
		console.log(`[RDP] canvas.drawImage #${_drawLogCount}: img=${img.constructor?.name || typeof img} ${img.width}x${img.height} at (${args[1]},${args[2]}), canvas=${(this as any).canvas?.width}x${(this as any).canvas?.height}`)
	}
	imgDrawn++
	return _origDrawImage.apply(this, args as any)
}

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
  console.log(`[RDP] updateContainerSize: window=${window.innerWidth}x${window.innerHeight} toolbarH=${toolbarH} fullscreen=${isFullscreen.value} container=${displayContainer.value.style.width}x${displayContainer.value.style.height}`)
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
  if (displayW === 0 || displayH === 0 || containerW === 0 || containerH === 0) {
    console.log(`[RDP] fitDisplay: SKIP zero dims container=${containerW}x${containerH} display=${displayW}x${displayH}`)
    return
  }
  const scale = Math.min(containerW / displayW, containerH / displayH)
  console.log(`[RDP] fitDisplay: container=${containerW}x${containerH} display=${displayW}x${displayH} scale=${scale}`)
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
    const stateNames: Record<number, string> = {
      [Guacamole.Tunnel.State.CONNECTING]: 'CONNECTING',
      [Guacamole.Tunnel.State.OPEN]: 'OPEN',
      [Guacamole.Tunnel.State.CLOSED]: 'CLOSED',
    }
    console.log('[RDP] Tunnel state:', tunnelState, stateNames[tunnelState] || 'UNKNOWN')
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
  guacDisplay.onresize = (width: number, height: number) => {
    console.log('[RDP] Display resized to:', width, 'x', height)
    fitDisplay()
  }

  // Diagnostic: count instructions and log on first sync
  guacClient.onsync = (timestamp: number, frames: number) => {
    syncCount++
    if (syncCount === 1) {
      // Scale display to fill container now that dimensions are known
      fitDisplay()
      console.log('[RDP] First sync! Instruction counts:', JSON.stringify(instrCounts))
      console.log('[RDP] Display dimensions:', guacDisplay.getWidth(), 'x', guacDisplay.getHeight())
      const canvas = displayEl.querySelector('canvas')
      if (canvas) {
        console.log('[RDP] First canvas:', canvas.width, 'x', canvas.height,
          'CSS:', canvas.style.width, canvas.style.height)
      }
      // Log all child elements of bounds div
      console.log('[RDP] Bounds children:', displayEl.children.length)
      for (let i = 0; i < displayEl.children.length; i++) {
        const child = displayEl.children[i] as HTMLElement
        console.log(`[RDP]   child ${i}:`, child.tagName, child.style.width, child.style.height)
      }
      // Canvas pixel sample — verify content was drawn after first sync
      const firstCanvas = displayEl.querySelector('canvas')
      if (firstCanvas) {
        try {
          const ctx = firstCanvas.getContext('2d')
          if (ctx) {
            const sample = ctx.getImageData(100, 100, 1, 1).data
            console.log(`[RDP] First canvas pixel (100,100): rgba(${sample[0]},${sample[1]},${sample[2]},${sample[3]})`)
          }
        } catch(e) { /* tainted */ }
      }
    }
    if (syncCount % 10 === 0) {
      console.log(`[RDP] Sync #${syncCount}, display:`, guacDisplay.getWidth(), 'x', guacDisplay.getHeight(),
        `imgDecoded=${imgDecoded} imgDrawn=${imgDrawn} imgFailed=${imgFailed}`)
      // Canvas pixel sample — check if content was actually drawn
      const canvas = displayEl.querySelector('canvas')
      if (canvas && canvas.width > 0 && canvas.height > 0) {
        try {
          const ctx = canvas.getContext('2d')
          if (ctx) {
            const sample = ctx.getImageData(
              Math.floor(canvas.width / 2), Math.floor(canvas.height / 2), 1, 1
            ).data
            console.log(`[RDP] Canvas center pixel: rgba(${sample[0]},${sample[1]},${sample[2]},${sample[3]}), canvasSize=${canvas.width}x${canvas.height}`)
          }
        } catch(e) { /* tainted canvas */ }
      }
    }
  }

  // Diagnostic: instruction counter (non-intrusive — wrap after Client sets it)
  const origOnInstr = tunnel.oninstruction
  tunnel.oninstruction = (opcode: string, args: string[]) => {
    instrCounts[opcode] = (instrCounts[opcode] || 0) + 1
    if (opcode === 'size') {
      console.log(`[RDP] size: layer=${args[0]} ${args[1]}x${args[2]}`)
    } else if (opcode === 'img') {
      console.log(`[RDP] img: stream=${args[0]} channelMask=${args[1]} layer=${args[2]} mime=${args[3]} ${args[4]}x${args[5]}`)
    } else if (syncCount === 0 && opcode !== 'blob' && opcode !== 'end') {
      console.log(`[RDP] pre-sync instr: ${opcode}`, args)
    }
    if (origOnInstr) origOnInstr(opcode, args)
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
