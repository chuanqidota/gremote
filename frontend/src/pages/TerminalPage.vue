<template>
  <div class="terminal-page" :style="pageStyle">
    <div class="terminal-toolbar" :style="toolbarStyle">
      <span class="connection-status" :style="{ color: statusColor }">
        {{ statusText }}
      </span>
      <div class="toolbar-right">
        <el-select
          v-model="currentTheme"
          size="small"
          style="width: 120px"
          @change="applyTheme"
        >
          <el-option label="Dark" value="dark" />
          <el-option label="Light" value="light" />
          <el-option label="Solarized" value="solarized-dark" />
          <el-option label="Dracula" value="dracula" />
        </el-select>
        <el-button size="small" @click="openFileBrowser">
          Files
        </el-button>
        <el-button size="small" @click="toggleFullscreen">
          {{ isFullscreen ? 'Exit' : 'Maximize' }}
        </el-button>
      </div>
    </div>
    <div ref="termContainer" class="terminal-container" />
    <FileListDialog
      v-model:visible="fileListVisible"
      :key="dialogKey"
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
import { AttachAddon } from 'xterm-addon-attach'
import 'xterm/css/xterm.css'
import { useWebSocket } from '../composables/useWebSocket'
import { useFileManager } from '../composables/useFileManager'
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
const wsHost = import.meta.env.VITE_WS_HOST || window.location.host

const termContainer = ref<HTMLDivElement>()
const fileListVisible = ref(false)
const uploadVisible = ref(false)
const dialogKey = ref(0)
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
    case 'connecting': return 'Connecting...'
    case 'connected': return 'Connected'
    case 'disconnected': return 'Disconnected'
    case 'error': return error.value || 'Error'
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
    ElMessage.error('Missing connection key')
    return
  }

  document.addEventListener('fullscreenchange', onFullscreenChange)

  const socket = connect(wsHost)

  socket.onopen = () => {
    nextTick(() => initTerminal(socket))
  }
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
  term.loadAddon(new AttachAddon(socket))
  term.open(termContainer.value)
  fitAddon.fit()
  term.focus()

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
  dialogKey.value++
}

onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  window.removeEventListener('resize', onResize)
  term?.dispose()
})
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
  padding: 8px 12px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
}

.connection-status {
  font-size: 13px;
  flex-shrink: 0;
}

.toolbar-right {
  display: flex;
  gap: 8px;
  margin-left: auto;
  align-items: center;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
}
</style>
