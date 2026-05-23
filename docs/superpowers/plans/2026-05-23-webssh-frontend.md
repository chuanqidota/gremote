# WebSSH Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Vue 3 + Element Plus + TypeScript frontend for the webssh-go backend, plus fix 4 backend issues.

**Architecture:** The frontend uses composables (useWebSocket, useFileManager, useAudit) to encapsulate backend communication. Pages are lazy-loaded via Vue Router. The API layer is a thin typed wrapper around axios. Backend fixes address a JSON tag typo, ES query injection, an IsConnected race condition, and route naming.

**Tech Stack:** Vue 3, TypeScript, Element Plus, Vue Router 4, xterm.js 5, asciinema-player, Axios, Vite

---

### Task 1: Backend fixes

**Files:**
- Modify: `app/api/params/params.go:8`
- Modify: `app/ws/utils/recordAudit/record.go:63-71`
- Modify: `pkg/redis/redis.go:80-87`
- Modify: `router/router.go:11-23`

- [ ] **Step 1: Fix JSON tag typo on Target field**

```go
// app/api/params/params.go, line 8
// Change json:"" to json:"target"
Target string `json:"target" comment:"目标地址" binding:"required"`
```

- [ ] **Step 2: Fix ES query injection in ReadData**

```go
// app/ws/utils/recordAudit/record.go, replace the ReadData function
// ReadData 从es中读取记录
func (e *EsRecord) ReadData(key string) []map[string]any {
	result := make([]map[string]any, 0)
	index := e.Index
	pageNum := 1
	pageSize := 10000
	for {
		from := (pageNum - 1) * pageSize
		query := map[string]any{
			"query": map[string]any{
				"bool": map[string]any{
					"must": []map[string]any{
						{"match": map[string]string{"key": key}},
					},
				},
			},
			"sort": []map[string]any{
				{"timeStamp": map[string]string{"order": "asc"}},
			},
			"from": from,
			"size": pageSize,
		}
		queryB, err := json.Marshal(query)
		if err != nil {
			return result
		}
		res, _ := es.Search(index, string(queryB))
		if len(res) == 0 {
			break
		}
		result = append(result, res...)
		pageNum++
	}
	return result
}
```

Add `"encoding/json"` to the imports in record.go.

- [ ] **Step 3: Fix IsConnected race condition with SetNX**

```go
// pkg/redis/redis.go, replace IsConnected function
// IsConnected 判断有没有连接过（原子操作，防止并发问题）
func IsConnected(key string) bool {
	isConnectedKey := key + "_connected"
	ok, err := RedisClient.SetNX(context.Background(), isConnectedKey, true, 24*60*60*time.Second).Result()
	if err != nil {
		logger.Error(fmt.Sprintf("SetNX失败-%s", err.Error()))
		return true // 出错时保守处理，阻止连接
	}
	return !ok // SetNX返回true=设置成功（未连接过），返回false=key已存在（已连接过）
}
```

- [ ] **Step 4: Fix route naming — split REST and WebSocket routes**

```go
// router/router.go, replace the Engine function
func Engine() *gin.Engine {
	router := gin.Default()

	api := router.Group("api/v1").Use(middleware.CORSMiddleware())
	{
		api.POST("obtain-key", api_view.ApiHandle.ObtainKey)
		api.GET("list-file", api_view.ApiHandle.ListFile)
		api.POST("upload-file", api_view.ApiHandle.UploadFile)
		api.GET("download-file", api_view.ApiHandle.DownLoadFile)
		api.GET("login-audit", api_view.ApiHandle.LoginAudit)
		api.GET("record-url", api_view.ApiHandle.RecordUrl)
	}

	ws := router.Group("ws/v1").Use(middleware.CORSMiddleware())
	{
		ws.GET(":key", ws_view.WsHandle.Handler)
	}

	return router
}
```

- [ ] **Step 5: Verify backend compiles**

```bash
cd /Users/zqqzqq/05_github/webssh-go && go build ./...
```

- [ ] **Step 6: Commit backend fixes**

```bash
git add app/api/params/params.go app/ws/utils/recordAudit/record.go pkg/redis/redis.go router/router.go
git commit -m "fix: JSON tag typo, ES query injection, IsConnected race, route naming"
```

---

### Task 2: Scaffold frontend project

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/tsconfig.json`
- Create: `frontend/tsconfig.node.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/index.html`
- Create: `frontend/.env.development`
- Create: `frontend/.env.production`
- Create: `frontend/env.d.ts`

- [ ] **Step 1: Create package.json**

```json
{
  "name": "webssh-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "axios": "^1.7.0",
    "element-plus": "^2.9.0",
    "vue": "^3.5.0",
    "vue-router": "^4.5.0",
    "xterm": "^5.3.0",
    "xterm-addon-attach": "^0.9.0",
    "xterm-addon-fit": "^0.8.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.0",
    "asciinema-player": "^3.8.0",
    "typescript": "~5.6.0",
    "vite": "^6.0.0",
    "vue-tsc": "^2.2.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "jsx": "preserve",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "noEmit": true,
    "paths": {
      "@/*": ["./src/*"]
    },
    "baseUrl": "."
  },
  "include": ["src/**/*.ts", "src/**/*.d.ts", "src/**/*.vue", "env.d.ts"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

- [ ] **Step 3: Create tsconfig.node.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "noEmit": true,
    "strict": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 4: Create vite.config.ts**

```ts
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8000',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 5: Create index.html**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>WebSSH</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 6: Create env.d.ts**

```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_WS_HOST: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
```

- [ ] **Step 7: Create .env.development**

```
VITE_WS_HOST=127.0.0.1:8000
```

- [ ] **Step 8: Create .env.production**

```
VITE_WS_HOST=window.location.host
```

- [ ] **Step 9: Install dependencies**

```bash
cd /Users/zqqzqq/05_github/webssh-go/frontend && npm install
```

- [ ] **Step 10: Commit**

```bash
git add frontend/
git commit -m "scaffold: create frontend project with Vite + Vue 3 + TypeScript"
```

---

### Task 3: Types and API layer

**Files:**
- Create: `frontend/src/types/index.ts`
- Create: `frontend/src/api/index.ts`

- [ ] **Step 1: Create types/index.ts**

```ts
export interface SSHInfo {
  target: string
  username: string
  password: string
  port: number
  user?: string
  source?: string
}

export interface FileItem {
  name: string
  size: number
  type: 'file' | 'directory'
}

export interface AuditRecord {
  key: string
  user: string
  source: string
  target: string
  startTime: string
  endTime: string
}

export interface AuditQuery {
  offset: number
  limit: number
  user?: string
  source?: string
  target?: string
  startTime?: string
  endTime?: string
  search?: string
}
```

- [ ] **Step 2: Create api/index.ts**

```ts
import axios from 'axios'
import type { SSHInfo, FileItem, AuditRecord, AuditQuery } from '../types'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

export async function obtainKey(info: SSHInfo): Promise<string> {
  const { data } = await http.post('/obtain-key', info)
  return data.key
}

export async function listFiles(key: string, path: string): Promise<FileItem[]> {
  const { data } = await http.get('/list-file', { params: { key, path } })
  return data.data
}

export async function uploadFile(key: string, path: string, file: File): Promise<void> {
  const form = new FormData()
  form.append('file', file)
  await http.post('/upload-file', form, { params: { key, path } })
}

export function getDownloadUrl(key: string, path: string, filename: string): string {
  return `/api/v1/download-file?key=${encodeURIComponent(key)}&path=${encodeURIComponent(path)}&filename=${encodeURIComponent(filename)}`
}

export async function queryAudit(query: AuditQuery): Promise<{ result: AuditRecord[]; count: number }> {
  const { data } = await http.get('/login-audit', { params: query })
  return data.data
}

export async function getRecordUrl(key: string): Promise<string> {
  const { data } = await http.get('/record-url', { params: { key } })
  return data.data
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/index.ts
git commit -m "feat: add TypeScript types and API layer"
```

---

### Task 4: Router and App shell

**Files:**
- Create: `frontend/src/router/index.ts`
- Create: `frontend/src/App.vue`

- [ ] **Step 1: Create router/index.ts**

```ts
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/connect',
  },
  {
    path: '/connect',
    name: 'connect',
    component: () => import('../pages/ConnectPage.vue'),
  },
  {
    path: '/term',
    name: 'term',
    component: () => import('../pages/TerminalPage.vue'),
  },
  {
    path: '/audit',
    name: 'audit',
    component: () => import('../pages/AuditPage.vue'),
  },
  {
    path: '/playback',
    name: 'playback',
    component: () => import('../pages/PlaybackPage.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
```

- [ ] **Step 2: Create App.vue**

```vue
<template>
  <router-view />
</template>
```

- [ ] **Step 3: Create placeholder pages directory**

```bash
mkdir -p /Users/zqqzqq/05_github/webssh-go/frontend/src/pages
mkdir -p /Users/zqqzqq/05_github/webssh-go/frontend/src/components
mkdir -p /Users/zqqzqq/05_github/webssh-go/frontend/src/composables
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/router/ frontend/src/App.vue
git commit -m "feat: add Vue Router and App shell"
```

---

### Task 5: useWebSocket composable

**Files:**
- Create: `frontend/src/composables/useWebSocket.ts`

- [ ] **Step 1: Create useWebSocket.ts**

```ts
import { ref, onUnmounted } from 'vue'

export type WsStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export function useWebSocket(key: string) {
  const status = ref<WsStatus>('connecting')
  const error = ref('')
  let socket: WebSocket | null = null

  function connect(host: string): WebSocket {
    const url = `ws://${host}/ws/v1/${key}`
    socket = new WebSocket(url)

    socket.onopen = () => {
      status.value = 'connected'
    }

    socket.onclose = () => {
      status.value = 'disconnected'
    }

    socket.onerror = () => {
      status.value = 'error'
      error.value = 'WebSocket connection failed'
    }

    return socket
  }

  function getSocket(): WebSocket | null {
    return socket
  }

  function close() {
    socket?.close()
    socket = null
  }

  onUnmounted(close)

  return { status, error, connect, getSocket, close }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/composables/useWebSocket.ts
git commit -m "feat: add useWebSocket composable"
```

---

### Task 6: useFileManager composable

**Files:**
- Create: `frontend/src/composables/useFileManager.ts`

- [ ] **Step 1: Create useFileManager.ts**

```ts
import { ref } from 'vue'
import { listFiles, uploadFile, getDownloadUrl } from '../api'
import type { FileItem } from '../types'

export function useFileManager(key: string) {
  const files = ref<FileItem[]>([])
  const loading = ref(false)
  const error = ref('')
  const currentPath = ref('/')

  async function fetchFiles(path: string) {
    loading.value = true
    error.value = ''
    try {
      files.value = await listFiles(key, path)
      currentPath.value = path
    } catch (e: any) {
      error.value = e?.response?.data?.msg || e?.message || 'Failed to list files'
    } finally {
      loading.value = false
    }
  }

  async function upload(file: File, path: string) {
    loading.value = true
    error.value = ''
    try {
      await uploadFile(key, path, file)
    } catch (e: any) {
      error.value = e?.response?.data?.msg || e?.message || 'Upload failed'
      throw e
    } finally {
      loading.value = false
    }
  }

  function download(path: string, filename: string) {
    const url = getDownloadUrl(key, path, filename)
    window.open(url, '_blank')
  }

  return { files, loading, error, currentPath, fetchFiles, upload, download }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/composables/useFileManager.ts
git commit -m "feat: add useFileManager composable"
```

---

### Task 7: useAudit composable

**Files:**
- Create: `frontend/src/composables/useAudit.ts`

- [ ] **Step 1: Create useAudit.ts**

```ts
import { ref } from 'vue'
import { queryAudit, getRecordUrl } from '../api'
import type { AuditRecord, AuditQuery } from '../types'

export function useAudit() {
  const data = ref<AuditRecord[]>([])
  const count = ref(0)
  const loading = ref(false)
  const error = ref('')

  async function fetch(query: AuditQuery) {
    loading.value = true
    error.value = ''
    try {
      const res = await queryAudit(query)
      data.value = res.result ?? []
      count.value = res.count ?? 0
    } catch (e: any) {
      error.value = e?.response?.data?.msg || e?.message || 'Failed to fetch audit records'
    } finally {
      loading.value = false
    }
  }

  async function fetchRecordUrl(key: string): Promise<string> {
    return getRecordUrl(key)
  }

  return { data, count, loading, error, fetch, fetchRecordUrl }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/composables/useAudit.ts
git commit -m "feat: add useAudit composable"
```

---

### Task 8: ConnectPage

**Files:**
- Create: `frontend/src/pages/ConnectPage.vue`

- [ ] **Step 1: Create ConnectPage.vue**

```vue
<template>
  <div class="connect-page">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      class="connect-form"
      status-icon
    >
      <h2>SSH Connection</h2>
      <el-form-item label="Host" prop="target">
        <el-input v-model="form.target" placeholder="192.168.1.1" />
      </el-form-item>
      <el-form-item label="Port" prop="port">
        <el-input v-model.number="form.port" placeholder="22" />
      </el-form-item>
      <el-form-item label="Username" prop="username">
        <el-input v-model="form.username" placeholder="root" />
      </el-form-item>
      <el-form-item label="Password" prop="password">
        <el-input v-model="form.password" type="password" show-password />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" :loading="loading" @click="onSubmit">
          Connect
        </el-button>
        <el-button @click="onReset">Reset</el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { obtainKey } from '../api'
import type { SSHInfo } from '../types'

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive<SSHInfo>({
  target: '',
  port: 22,
  username: '',
  password: '',
})

const rules: FormRules = {
  target: [{ required: true, message: 'Host is required', trigger: 'blur' }],
  port: [{ required: true, message: 'Port is required', trigger: 'blur' }],
  username: [{ required: true, message: 'Username is required', trigger: 'blur' }],
  password: [{ required: true, message: 'Password is required', trigger: 'blur' }],
}

async function onSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const key = await obtainKey(form)
    window.open(`/term?key=${key}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.msg || e?.message || 'Connection failed')
  } finally {
    loading.value = false
  }
}

function onReset() {
  formRef.value?.resetFields()
}
</script>

<style scoped>
.connect-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f7fa;
}

.connect-form {
  width: 420px;
  padding: 32px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.connect-form h2 {
  text-align: center;
  margin-bottom: 24px;
  color: #303133;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/ConnectPage.vue
git commit -m "feat: add ConnectPage with SSH form and validation"
```

---

### Task 9: TerminalPage

**Files:**
- Create: `frontend/src/pages/TerminalPage.vue`

- [ ] **Step 1: Create TerminalPage.vue**

```vue
<template>
  <div class="terminal-page">
    <div class="terminal-toolbar">
      <el-button size="small" @click="fileListVisible = true">
        Browse Files
      </el-button>
      <el-button size="small" @click="uploadVisible = true">
        Upload
      </el-button>
      <el-button size="small" @click="openFileBrowser">
        Files
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
    rendererType: 'canvas',
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
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/TerminalPage.vue
git commit -m "feat: add TerminalPage with xterm and file toolbar"
```

---

### Task 10: FileListDialog

**Files:**
- Create: `frontend/src/components/FileListDialog.vue`

- [ ] **Step 1: Create FileListDialog.vue**

```vue
<template>
  <el-dialog
    :model-value="visible"
    title="Browse Files"
    width="640px"
    @update:model-value="$emit('update:visible', $event)"
    @open="onOpen"
  >
    <div class="file-path-bar">
      <el-button size="small" @click="goUp" :disabled="currentPath === '/'">
        Up
      </el-button>
      <el-input :model-value="currentPath" readonly size="small" />
    </div>
    <el-table
      :data="files"
      v-loading="loading"
      max-height="400"
      @row-click="onRowClick"
      style="cursor: pointer"
    >
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="size" label="Size" width="120" />
      <el-table-column prop="type" label="Type" width="100">
        <template #default="{ row }">
          {{ row.type === 'directory' ? 'Folder' : 'File' }}
        </template>
      </el-table-column>
      <el-table-column label="Action" width="100">
        <template #default="{ row }">
          <el-button
            v-if="row.type === 'file'"
            size="small"
            type="primary"
            link
            @click.stop="onDownload(row)"
          >
            Download
          </el-button>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FileItem } from '../types'
import type { useFileManager } from '../composables/useFileManager'

const props = defineProps<{
  visible: boolean
  fileManager: ReturnType<typeof useFileManager>
}>()

defineEmits<{
  'update:visible': [value: boolean]
}>()

const files = computed(() => props.fileManager.files.value)
const loading = computed(() => props.fileManager.loading.value)
const currentPath = computed(() => props.fileManager.currentPath.value)

function onOpen() {
  props.fileManager.fetchFiles('/')
}

function onRowClick(row: FileItem) {
  if (row.type === 'directory') {
    props.fileManager.fetchFiles(currentPath.value + '/' + row.name)
  }
}

function goUp() {
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  const parent = '/' + parts.join('/')
  props.fileManager.fetchFiles(parent || '/')
}

function onDownload(row: FileItem) {
  const path = currentPath.value
  props.fileManager.download(path, row.name)
}
</script>

<style scoped>
.file-path-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: center;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/FileListDialog.vue
git commit -m "feat: add FileListDialog for SFTP file browsing"
```

---

### Task 11: FileUploadDialog

**Files:**
- Create: `frontend/src/components/FileUploadDialog.vue`

- [ ] **Step 1: Create FileUploadDialog.vue**

```vue
<template>
  <el-dialog
    :model-value="visible"
    title="Upload File"
    width="480px"
    @update:model-value="$emit('update:visible', $event)"
  >
    <el-form label-width="80px">
      <el-form-item label="Target Path">
        <el-input v-model="uploadPath" placeholder="/tmp" />
      </el-form-item>
      <el-form-item label="File">
        <el-upload
          :auto-upload="false"
          :limit="1"
          :on-change="onFileChange"
          :file-list="fileList"
        >
          <el-button type="primary">Select File</el-button>
        </el-upload>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">Cancel</el-button>
      <el-button
        type="primary"
        :loading="loading"
        :disabled="!selectedFile"
        @click="onUpload"
      >
        Upload
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, type UploadFile } from 'element-plus'
import type { useFileManager } from '../composables/useFileManager'

const props = defineProps<{
  visible: boolean
  fileManager: ReturnType<typeof useFileManager>
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  uploaded: []
}>()

const uploadPath = ref('/')
const selectedFile = ref<File | null>(null)
const fileList = ref<UploadFile[]>([])
const loading = ref(false)

function onFileChange(file: UploadFile) {
  selectedFile.value = file.raw ?? null
}

async function onUpload() {
  if (!selectedFile.value) return
  loading.value = true
  try {
    await props.fileManager.upload(selectedFile.value, uploadPath.value)
    ElMessage.success('Upload complete')
    emit('uploaded')
    emit('update:visible', false)
    selectedFile.value = null
    fileList.value = []
  } catch (e: any) {
    ElMessage.error(e?.message || 'Upload failed')
  } finally {
    loading.value = false
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/FileUploadDialog.vue
git commit -m "feat: add FileUploadDialog for SFTP file upload"
```

---

### Task 12: AuditPage

**Files:**
- Create: `frontend/src/pages/AuditPage.vue`

- [ ] **Step 1: Create AuditPage.vue**

```vue
<template>
  <div class="audit-page">
    <h2>Login Audit</h2>

    <div class="audit-filters">
      <el-input
        v-model="search"
        placeholder="Search user, source, or target..."
        clearable
        style="width: 280px"
        @input="onSearch"
      />
      <el-date-picker
        v-model="dateRange"
        type="datetimerange"
        range-separator="to"
        start-placeholder="Start"
        end-placeholder="End"
        format="YYYY-MM-DD HH:mm:ss"
        value-format="YYYY-MM-DD HH:mm:ss"
        @change="onDateChange"
      />
      <el-button type="primary" @click="fetchData">Query</el-button>
    </div>

    <el-table :data="data" v-loading="loading" border stripe style="margin-top: 16px">
      <el-table-column prop="user" label="User" width="120" />
      <el-table-column prop="source" label="Source" width="160" />
      <el-table-column prop="target" label="Target" width="160" />
      <el-table-column prop="startTime" label="Start Time" width="180" />
      <el-table-column prop="endTime" label="End Time" width="180" />
      <el-table-column prop="key" label="Key" min-width="200" show-overflow-tooltip />
      <el-table-column label="Actions" width="100" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="onPlayback(row.key)">
            Playback
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="count"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      style="margin-top: 16px; justify-content: flex-end"
      @size-change="fetchData"
      @current-change="fetchData"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAudit } from '../composables/useAudit'

const { data, count, loading, fetch } = useAudit()

const search = ref('')
const dateRange = ref<[string, string] | null>(null)
const page = ref(1)
const pageSize = ref(10)

let searchTimer: ReturnType<typeof setTimeout> | null = null

function buildQuery() {
  return {
    offset: (page.value - 1) * pageSize.value,
    limit: pageSize.value,
    search: search.value || undefined,
    startTime: dateRange.value?.[0],
    endTime: dateRange.value?.[1],
  }
}

function fetchData() {
  fetch(buildQuery())
}

function onSearch() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    fetchData()
  }, 300)
}

function onDateChange() {
  page.value = 1
}

function onPlayback(key: string) {
  window.open(`/playback?key=${key}`, '_blank')
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.audit-page {
  padding: 24px;
  background: #f5f7fa;
  min-height: 100vh;
}

.audit-page h2 {
  margin-bottom: 16px;
  color: #303133;
}

.audit-filters {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/AuditPage.vue
git commit -m "feat: add AuditPage with search, date filter, and pagination"
```

---

### Task 13: PlaybackPage

**Files:**
- Create: `frontend/src/pages/PlaybackPage.vue`

- [ ] **Step 1: Create PlaybackPage.vue**

```vue
<template>
  <div class="playback-page">
    <div v-if="loading" class="loading">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>Loading recording...</span>
    </div>
    <div v-else-if="error" class="error">
      <p>{{ error }}</p>
      <el-button @click="$router.back()">Go Back</el-button>
    </div>
    <div v-else ref="playerContainer" class="player-container" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { Loading } from '@element-plus/icons-vue'
import * as AsciinemaPlayer from 'asciinema-player'
import 'asciinema-player/dist/bundle/asciinema-player.css'
import { useAudit } from '../composables/useAudit'

const route = useRoute()
const key = route.query.key as string

const { fetchRecordUrl } = useAudit()
const playerContainer = ref<HTMLDivElement>()
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  if (!key) {
    error.value = 'Missing key parameter'
    loading.value = false
    return
  }

  try {
    const url = await fetchRecordUrl(key)
    await nextTick()
    if (playerContainer.value) {
      AsciinemaPlayer.create(url, playerContainer.value, {
        autoPlay: true,
        speed: 1.0,
        idleTimeLimit: 2,
      })
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load recording'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.playback-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #1e1e1e;
}

.loading,
.error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #ccc;
  font-size: 16px;
}

.player-container {
  width: 100%;
  max-width: 960px;
}
</style>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/PlaybackPage.vue
git commit -m "feat: add PlaybackPage with asciinema player"
```

---

### Task 14: Wire up main.ts

**Files:**
- Create: `frontend/src/main.ts`

- [ ] **Step 1: Create main.ts**

```ts
import { createApp } from 'vue'
import App from './App.vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import router from './router'

const app = createApp(App)
app.use(ElementPlus)
app.use(router)
app.mount('#app')
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/main.ts
git commit -m "feat: wire up main.ts with Element Plus and Router"
```

---

### Task 15: Verify the build

- [ ] **Step 1: Build the frontend**

```bash
cd /Users/zqqzqq/05_github/webssh-go/frontend && npm run build
```

Expected: Build succeeds with no TypeScript errors.

- [ ] **Step 2: Fix any type errors and recommit if needed**

---

### Task 16: Final integration check

- [ ] **Step 1: Verify backend still compiles with route changes**

```bash
cd /Users/zqqzqq/05_github/webssh-go && go build ./...
```

- [ ] **Step 2: Verify all files in place**

```bash
ls -la /Users/zqqzqq/05_github/webssh-go/frontend/src/pages/
ls -la /Users/zqqzqq/05_github/webssh-go/frontend/src/components/
ls -la /Users/zqqzqq/05_github/webssh-go/frontend/src/composables/
```

- [ ] **Step 3: Commit any remaining changes**

```bash
git add -A
git status
```
