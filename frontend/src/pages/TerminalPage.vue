<template>
  <div class="terminal-page" :style="pageStyle">
    <div class="terminal-toolbar" :style="toolbarStyle">
      <span class="connection-status" :style="{ color: statusColor }">
        ● {{ statusText }}
      </span>
      <span v-if="hostIp" class="host-info" :style="{ color: '#909399' }">
        {{ hostIp }}
      </span>
      <div class="toolbar-right">
        <el-select
          v-model="currentTheme"
          size="small"
          style="width: 100px"
          @change="applyTheme"
        >
          <el-option label="暗色" value="dark" />
          <el-option label="亮色" value="light" />
          <el-option label="日光" value="solarized-dark" />
          <el-option label="德古拉" value="dracula" />
        </el-select>
        <el-button size="small" @click="openFileBrowser">
          文件
        </el-button>
        <el-button size="small" :title="isFullscreen ? '退出全屏' : '全屏'" @click="toggleFullscreen">
          <el-icon><FullScreen v-if="!isFullscreen" /><Close v-else /></el-icon>
        </el-button>
      </div>
    </div>
    <div ref="termContainer" class="terminal-container" />
    <FileListDialog
      v-model:visible="fileListVisible"
      :file-manager="fileManager"
      @upload="uploadVisible = true"
    />
    <FileUploadDialog
      v-model:visible="uploadVisible"
      :file-manager="fileManager"
      @uploaded="onFileUploaded"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import { useWebSocket } from '../composables/useWebSocket'
import { useFileManager } from '../composables/useFileManager'
import { FullScreen, Close } from '@element-plus/icons-vue'
import FileListDialog from '../components/FileListDialog.vue'
import FileUploadDialog from '../components/FileUploadDialog.vue'

const themes: Record<string, { fg: string; bg: string; cursor: string; selection: string; toolbar: string; page: string; border: string }> = {
  dark: {
    fg: '#d4d4d4',
    bg: '#1e1e1e',
    cursor: '#ffffff',
    selection: '#264f78',
    toolbar: '#2d2d2d',
    page: '#1e1e1e',
    border: '#3d3d3d',
  },
  light: {
    fg: '#333333',
    bg: '#ffffff',
    cursor: '#000000',
    selection: '#add6ff',
    toolbar: '#f0f0f0',
    page: '#ffffff',
    border: '#d9d9d9',
  },
  'solarized-dark': {
    fg: '#839496',
    bg: '#002b36',
    cursor: '#93a1a1',
    selection: '#073642',
    toolbar: '#073642',
    page: '#002b36',
    border: '#586e75',
  },
  dracula: {
    fg: '#f8f8f2',
    bg: '#282a36',
    cursor: '#f8f8f2',
    selection: '#44475a',
    toolbar: '#21222c',
    page: '#282a36',
    border: '#6272a4',
  },
}

const route = useRoute()
const key = route.query.key as string
const hostIp = route.query.host as string
const wsHost = import.meta.env.VITE_WS_HOST || window.location.host

const termContainer = ref<HTMLDivElement>()
const fileListVisible = ref(false)
const uploadVisible = ref(false)
const currentTheme = ref('dark')

const { status, error, connect, getSocket } = useWebSocket(key)
const fileManager = useFileManager(key)

let term: Terminal | null = null
let fitAddon: FitAddon | null = null

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
    case 'connecting': return ''
    case 'connected': return '已连接'
    case 'disconnected': return '已断开'
    case 'error': return error.value || '错误'
    default: return ''
  }
})

const pageStyle = computed(() => ({
  '--bg-page': themes[currentTheme.value].page,
}))

const toolbarStyle = computed(() => ({
  background: themes[currentTheme.value].toolbar,
  borderColor: themes[currentTheme.value].border,
}))

const isFullscreen = ref(false)

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
    nextTick(() => initTerminal(socket))
  })
})

function initTerminal(socket: WebSocket) {
  if (!termContainer.value) return

  term = new Terminal({
    fontSize: 14,
    cursorBlink: true,
    disableStdin: false,
    theme: {
      foreground: themes[currentTheme.value].fg,
      background: themes[currentTheme.value].bg,
      cursor: themes[currentTheme.value].cursor,
      selectionBackground: themes[currentTheme.value].selection,
    },
  })

  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termContainer.value)
  fitAddon.fit()
  term.focus()

  // 手动处理 WebSocket 双向通信（替代 AttachAddon）
  term.onData((data) => {
    if (socket.readyState === WebSocket.OPEN) {
      socket.send(data)
    }
  })

  socket.addEventListener('message', (ev) => {
    const data = typeof ev.data === 'string' ? ev.data : new Uint8Array(ev.data)
    term?.write(data)
  })

  socket.send(JSON.stringify({ resize: [term.cols, term.rows] }))

  window.addEventListener('resize', onResize)
}

function onResize() {
  if (!term || !fitAddon) return
  fitAddon.fit()
  const socket = getSocket()
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ resize: [term.cols, term.rows] }))
  }
}

function applyTheme(name: string) {
  if (term) {
    const t = themes[name]
    term.options.theme = {
      foreground: t.fg,
      background: t.bg,
      cursor: t.cursor,
      selectionBackground: t.selection,
    }
  }
}

function openFileBrowser() {
  fileListVisible.value = true
  fileManager.fetchFiles(fileManager.currentPath.value)
}

function onFileUploaded() {
  uploadVisible.value = false
  fileManager.fetchFiles(fileManager.currentPath.value)
}
</script>

<style scoped>
.terminal-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--bg-page, #1e1e1e);
}

.terminal-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
}

.connection-status {
  font-size: 13px;
  flex-shrink: 0;
}

.host-info {
  font-size: 11px;
}

.toolbar-right {
  display: flex;
  gap: 6px;
  margin-left: auto;
  align-items: center;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
}
</style>
