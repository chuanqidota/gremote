<template>
  <div class="terminal-page">
    <div class="terminal-toolbar">
      <el-button size="small" @click="openFileBrowser">
        Files
      </el-button>
      <el-button size="small" @click="uploadVisible = true">
        Upload
      </el-button>
    </div>
    <div ref="termContainer" class="terminal-container" />
    <FileListDialog
      v-model:visible="fileListVisible"
      :key="dialogKey"
      :file-manager="fileManager"
    />
    <FileUploadDialog
      v-model:visible="uploadVisible"
      :file-manager="fileManager"
      @uploaded="onFileUploaded"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
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

const route = useRoute()
const key = route.query.key as string
const wsHost = import.meta.env.VITE_WS_HOST

const termContainer = ref<HTMLDivElement>()
const fileListVisible = ref(false)
const uploadVisible = ref(false)
const dialogKey = ref(0)

const { status, error, connect, getSocket } = useWebSocket(key)
const fileManager = useFileManager(key)

let term: Terminal | null = null
let fitAddon: FitAddon | null = null

onMounted(() => {
  if (!key) {
    ElMessage.error('Missing connection key')
    return
  }

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

function openFileBrowser() {
  fileListVisible.value = true
  fileManager.fetchFiles(fileManager.currentPath.value)
}

function onFileUploaded() {
  uploadVisible.value = false
  dialogKey.value++
}

onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  term?.dispose()
})
</script>

<style scoped>
.terminal-page {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1e1e1e;
}

.terminal-toolbar {
  display: flex;
  gap: 8px;
  padding: 8px 12px;
  background: #2d2d2d;
  border-bottom: 1px solid #3d3d3d;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
}
</style>
