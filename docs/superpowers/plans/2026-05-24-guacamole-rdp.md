# Guacamole RDP Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Windows RDP remote desktop support using guacd + guacamole-common-js, with session recording stored in MinIO + ES.

**Architecture:** Go backend communicates with guacd via Guacamole protocol over TCP. Browser connects to Go backend via WebSocket, which bridges to guacd. guacd handles RDP protocol and screen rendering. All guacd instructions are recorded as .guac files in MinIO.

**Tech Stack:** Go (Gin, gorilla/websocket), guacd (Docker), Vue 3 + guacamole-common-js, Redis, Elasticsearch, MinIO

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `backend/pkg/guacamole/client.go` | Guacamole protocol TCP client — connect, read/write instructions |
| `backend/pkg/guacamole/protocol.go` | Instruction encoding/decoding helpers |
| `backend/app/ws/view/rdp.go` | RDP WebSocket handler — bridges browser WS ↔ guacd |
| `backend/app/api/params/rdp.go` | `RDPInfo` struct for RDP connection parameters |

### Modified Files

| File | Change |
|------|--------|
| `backend/config/config.go` | Add `GuacdConfig` struct |
| `backend/config/config.yaml` | Add `guacd:` section |
| `backend/app/api/view/view.go` | Add `ObtainKeyRDP` handler |
| `backend/router/router.go` | Add RDP API + WS routes |
| `backend/app/ws/utils/loginAudit/audit.go` | Support RDP login audit (protocol field) |
| `backend/app/ws/utils/recordAudit/record.go` | Support RDP record audit |
| `docker-compose.yaml` | Add guacd service |
| `frontend/src/types/index.ts` | Add `RDPInfo` interface |
| `frontend/src/api/index.ts` | Add `obtainKeyRDP()` function |
| `frontend/src/router/index.ts` | Add `/rdp` route |
| `frontend/src/pages/ConnectPage.vue` | Add Linux/Windows tab switching |
| `frontend/index.html` | Add guacamole-common-js script (or npm install) |

### New Frontend Files

| File | Responsibility |
|------|---------------|
| `frontend/src/pages/RdpPage.vue` | RDP remote desktop page with guacamole-common-js |
| `frontend/src/composables/useRdpWebSocket.ts` | WebSocket composable for RDP endpoint |

---

## Task 1: Config + Docker Setup

**Files:**
- Modify: `backend/config/config.go`
- Modify: `backend/config/config.yaml`
- Modify: `docker-compose.yaml`

- [ ] **Step 1: Add GuacdConfig struct to config.go**

Read `backend/config/config.go`. Add a new struct after `S3Config`:

```go
type GuacdConfig struct {
    Host string `yaml:"host" comment:"guacd主机地址"`
    Port int    `yaml:"port" comment:"guacd端口"`
}
```

Add the field to the `Config` struct:

```go
type Config struct {
    Server    ServerConfig    `yaml:"server"`
    Redis     RedisConfig     `yaml:"redis"`
    ES        ESConfig        `yaml:"elasticsearch"`
    Audit     AuditConfig     `yaml:"audit"`
    S3        S3Config        `yaml:"s3"`
    Guacd     GuacdConfig     `yaml:"guacd"`
}
```

- [ ] **Step 2: Add guacd section to config.yaml**

Read `backend/config/config.yaml`. Add at the end:

```yaml
guacd:
  host: guacd
  port: 4822
```

- [ ] **Step 3: Add guacd service to docker-compose.yaml**

Read `docker-compose.yaml`. Add a new service after `redis`:

```yaml
  guacd:
    image: guacamole/guacd
    restart: always
    ports:
      - "4822:4822"
```

- [ ] **Step 4: Verify backend compiles**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/config/config.go backend/config/config.yaml docker-compose.yaml
git commit -m "feat: add guacd config and docker service for RDP support"
```

---

## Task 2: Guacamole Protocol Client

**Files:**
- Create: `backend/pkg/guacamole/protocol.go`
- Create: `backend/pkg/guacamole/client.go`

- [ ] **Step 1: Create protocol.go with instruction encoding/decoding**

```go
package guacamole

import (
    "fmt"
    "io"
    "strconv"
    "strings"
)

// Instruction represents a Guacamole protocol instruction.
// Format: length.field1,length.field2,...,length.fieldN;
type Instruction struct {
    Op    string
    Args  []string
}

// WriteInstruction encodes an instruction and writes it to the writer.
func WriteInstruction(w io.Writer, op string, args ...string) error {
    parts := make([]string, 0, len(args)+1)
    parts = append(parts, encodeField(op))
    for _, arg := range args {
        parts = append(parts, encodeField(arg))
    }
    _, err := io.WriteString(w, strings.Join(parts, ",")+";")
    return err
}

func encodeField(s string) string {
    return fmt.Sprintf("%d.%s", len(s), s)
}

// ReadInstruction reads one instruction from the reader.
func ReadInstruction(r io.Reader) (*Instruction, error) {
    // Read until ';' delimiter
    var buf []byte
    one := make([]byte, 1)
    for {
        n, err := r.Read(one)
        if err != nil {
            return nil, err
        }
        if n == 0 {
            continue
        }
        if one[0] == ';' {
            break
        }
        buf = append(buf, one[0])
    }

    raw := string(buf)
    if raw == "" {
        return nil, fmt.Errorf("empty instruction")
    }

    // Parse fields: "5.abc,3.def;"
    fields := strings.Split(raw, ",")
    instr := &Instruction{}
    for i, field := range fields {
        dotIdx := strings.Index(field, ".")
        if dotIdx < 0 {
            continue
        }
        length, err := strconv.Atoi(field[:dotIdx])
        if err != nil {
            continue
        }
        value := field[dotIdx+1:]
        if len(value) > length {
            value = value[:length]
        }
        if i == 0 {
            instr.Op = value
        } else {
            instr.Args = append(instr.Args, value)
        }
    }
    return instr, nil
}
```

- [ ] **Step 2: Create client.go with Guacamole client**

```go
package guacamole

import (
    "fmt"
    "net"
    "sync"
)

// Client manages a connection to guacd.
type Client struct {
    conn    net.Conn
    mu      sync.Mutex
    closed  bool
}

// Connect creates a new TCP connection to guacd.
func Connect(host string, port int) (*Client, error) {
    addr := fmt.Sprintf("%s:%d", host, port)
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to guacd at %s: %w", addr, err)
    }
    return &Client{conn: conn}, nil
}

// Handshake sends the Guacamole protocol handshake (select + connection params).
func (c *Client) Handshake(protocol string, params map[string]string) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Send protocol selection
    if err := WriteInstruction(c.conn, "select", protocol); err != nil {
        return err
    }

    // Read server version response
    _, err := ReadInstruction(c.conn)
    if err != nil {
        return fmt.Errorf("failed to read server version: %w", err)
    }

    // Send connection parameters
    for key, value := range params {
        if err := WriteInstruction(c.conn, key, value); err != nil {
            return err
        }
    }

    // Send ready signal
    if err := WriteInstruction(c.conn, "ready"); err != nil {
        return err
    }

    // Read connection parameters from server
    for {
        instr, err := ReadInstruction(c.conn)
        if err != nil {
            return fmt.Errorf("failed to read handshake response: %w", err)
        }
        if instr.Op == "ready" {
            break
        }
        // Server may send supported instructions list
    }

    return nil
}

// Write sends an instruction to guacd.
func (c *Client) Write(op string, args ...string) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.closed {
        return fmt.Errorf("client is closed")
    }
    return WriteInstruction(c.conn, op, args...)
}

// Read reads one instruction from guacd.
func (c *Client) Read() (*Instruction, error) {
    return ReadInstruction(c.conn)
}

// Conn returns the underlying net.Conn for direct I/O (used for streaming).
func (c *Client) Conn() net.Conn {
    return c.conn
}

// Close closes the connection to guacd.
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    if c.closed {
        return nil
    }
    c.closed = true
    return c.conn.Close()
}
```

- [ ] **Step 3: Verify backend compiles**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add backend/pkg/guacamole/
git commit -m "feat: add Guacamole protocol client for guacd communication"
```

---

## Task 3: RDP Params + API Handler

**Files:**
- Create: `backend/app/api/params/rdp.go`
- Modify: `backend/app/api/view/view.go`
- Modify: `backend/router/router.go`

- [ ] **Step 1: Create RDPInfo struct**

Create `backend/app/api/params/rdp.go`:

```go
package params

type RDPInfo struct {
    User     string `json:"user"`
    Source   string `json:"source"`
    Target   string `json:"target" binding:"required"`
    Port     int    `json:"port" binding:"required"`
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
    Domain   string `json:"domain"`
}
```

- [ ] **Step 2: Add ObtainKeyRDP handler**

Read `backend/app/api/view/view.go`. Add method to `apiHandle`:

```go
// ObtainKeyRDP generates a one-time key for RDP connection
func (a *apiHandle) ObtainKeyRDP(c *gin.Context) {
    var info params.RDPInfo
    if err := c.ShouldBindJSON(&info); err != nil {
        response.Fail(c, "参数错误: "+err.Error())
        return
    }
    if info.Port == 0 {
        info.Port = 3389
    }
    info.Source = c.ClientIP()
    key := uuid.New().String()
    if err := redis.Set(key, info, config.Conf.Server.SessionTTL); err != nil {
        response.Fail(c, "存储失败")
        return
    }
    response.KeyRes(c, key)
}
```

Add the import for `params` package if not already imported.

- [ ] **Step 3: Register RDP routes**

Read `backend/router/router.go`. Add routes:

```go
// In the api/v1 group:
api.POST("obtain-key-rdp", apiview.ApiHandle.ObtainKeyRDP)

// In the ws/v1 group:
ws.GET("rdp/:key", wsview.WsHandle.RDPHandler)
```

- [ ] **Step 4: Verify backend compiles**

Run: `cd backend && go build ./...`
Expected: no errors (RDpHandler not implemented yet, will be a placeholder)

- [ ] **Step 5: Commit**

```bash
git add backend/app/api/params/rdp.go backend/app/api/view/view.go backend/router/router.go
git commit -m "feat: add RDP params, API handler, and routes"
```

---

## Task 4: RDP WebSocket Handler

**Files:**
- Create: `backend/app/ws/view/rdp.go`

- [ ] **Step 1: Create RDP WebSocket handler**

Create `backend/app/ws/view/rdp.go`:

```go
package view

import (
    "fmt"
    "io"
    "net/http"
    "time"

    "gwebssh/app/api/params"
    "gwebssh/app/ws/utils/loginAudit"
    "gwebssh/app/ws/utils/recordAudit"
    "gwebssh/config"
    "gwebssh/pkg/guacamole"
    "gwebssh/pkg/logger"
    "gwebssh/pkg/redis"
    "gwebssh/pkg/s3"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

func (w wsHandle) RDPHandler(c *gin.Context) {
    // Upgrade HTTP to WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    // Validate key
    key := c.Param("key")
    if key == "" {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("无效链接"))
        return
    }
    if redis.IsConnected(key) {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("链接失效,已经被链接过一次"))
        return
    }

    // Retrieve RDP info from Redis
    var info params.RDPInfo
    if err := redis.Get(key, &info); err != nil {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("获取登录信息失败"))
        return
    }

    // Auto-detect client IP
    clientIP := c.ClientIP()
    if info.User == "" {
        info.User = clientIP
    }
    if info.Source == "" {
        info.Source = clientIP
    }

    // Write login audit to ES
    e := loginAudit.NewEsAudit()
    defer redis.DeleteKey(key)
    auditData := map[string]any{
        "key":       key,
        "startTime": time.Now().Format("2006-01-02 15:04:05"),
        "user":      info.User,
        "source":    info.Source,
        "target":    info.Target,
        "protocol":  "rdp",
    }
    e.WriteData(auditData)
    defer e.UpdateEndTime(key)

    // Read first message for initial size
    _, firstMessage, _ := conn.ReadMessage()
    var sizeMsg map[string][]int
    if err := json.Unmarshal(firstMessage, &sizeMsg); err != nil {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("接收窗口大小失败"))
        return
    }
    sizeData, ok := sizeMsg["resize"]
    if !ok || len(sizeData) < 2 {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("窗口大小数据格式错误"))
        return
    }
    width := sizeData[0]
    height := sizeData[1]

    // Connect to guacd
    guacClient, err := guacamole.Connect(
        config.Conf.Guacd.Host,
        config.Conf.Guacd.Port,
    )
    if err != nil {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("连接guacd失败: "+err.Error()))
        return
    }
    defer guacClient.Close()

    // Guacamole handshake
    params := map[string]string{
        "hostname":   info.Target,
        "port":       fmt.Sprintf("%d", info.Port),
        "username":   info.Username,
        "password":   info.Password,
        "security":   "any",
        "ignore-cert": "true",
    }
    if info.Domain != "" {
        params["domain"] = info.Domain
    }
    if err := guacClient.Handshake("rdp", params); err != nil {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("guacd握手失败: "+err.Error()))
        return
    }

    // Send initial size
    if err := guacClient.Write("size", fmt.Sprintf("%d", width), fmt.Sprintf("%d", height)); err != nil {
        _ = conn.WriteMessage(websocket.TextMessage, []byte("发送窗口大小失败"))
        return
    }

    // Initialize recording
    record := recordAudit.NewEsRecord()
    recordData := map[string]any{
        "key":       key,
        "timeStamp": time.Now().UnixNano() / int64(time.Millisecond),
        "history":   fmt.Sprintf(`[0,"o","%s"]`, fmt.Sprintf("rdp-session-%s", key)),
    }
    record.WriteData(recordData)

    // Start recording writer goroutine
    recordingData := make(chan []byte, 1024)
    quitChan := make(chan bool, 4)

    // Goroutine 1: WebSocket → guacd (user input)
    go func() {
        defer func() { quitChan <- true }()
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                return
            }
            // Parse JSON instruction from browser
            var msg map[string]any
            if err := json.Unmarshal(message, &msg); err != nil {
                continue
            }
            op, ok := msg["op"].(string)
            if !ok {
                continue
            }
            args := make([]string, 0)
            if argsArr, ok := msg["args"].([]any); ok {
                for _, a := range argsArr {
                    if s, ok := a.(string); ok {
                        args = append(args, s)
                    }
                }
            }
            if err := guacClient.Write(op, args...); err != nil {
                return
            }
        }
    }()

    // Goroutine 2: guacd → WebSocket + recording channel
    go func() {
        defer func() { quitChan <- true }()
        for {
            instr, err := guacClient.Read()
            if err != nil {
                return
            }
            // Forward to browser as JSON
            msg := map[string]any{
                "op":   instr.Op,
                "args": instr.Args,
            }
            data, _ := json.Marshal(msg)
            _ = conn.WriteMessage(websocket.TextMessage, data)

            // Send to recording channel
            recordingData <- data
        }
    }()

    // Goroutine 3: recording channel → MinIO
    go func() {
        defer func() { quitChan <- true }()
        var recordingBuf []byte
        for {
            select {
            case <-quitChan:
                // Upload final recording
                if len(recordingBuf) > 0 {
                    _ = s3.UploadFile(key, recordingBuf)
                }
                return
            case data := <-recordingBuf:
                recordingBuf = append(recordingBuf, data...)
            }
        }
    }()

    // Wait for session to end
    <-quitChan
}
```

Note: The recording goroutine needs a small fix — `recordingBuf` channel should be `recordingData`. Let me correct that:

```go
    // Goroutine 3: recording channel → MinIO
    go func() {
        defer func() { quitChan <- true }()
        var recordingBuf []byte
        for {
            select {
            case <-quitChan:
                if len(recordingBuf) > 0 {
                    _ = s3.UploadFile(key, recordingBuf)
                }
                return
            case data := <-recordingData:
                recordingBuf = append(recordingBuf, data...)
            }
        }
    }()
```

- [ ] **Step 2: Verify backend compiles**

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add backend/app/ws/view/rdp.go
git commit -m "feat: add RDP WebSocket handler with guacd bridge and recording"
```

---

## Task 5: Frontend Types + API

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/index.ts`

- [ ] **Step 1: Add RDPInfo type**

Read `frontend/src/types/index.ts`. Add:

```typescript
export interface RDPInfo {
  target: string
  port: number
  username: string
  password: string
  domain?: string
}
```

- [ ] **Step 2: Add obtainKeyRDP API function**

Read `frontend/src/api/index.ts`. Add:

```typescript
export function obtainKeyRDP(info: RDPInfo): Promise<{ key: string }> {
  return request.post('/api/v1/obtain-key-rdp', info)
}
```

Add the import for `RDPInfo` from types.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/index.ts
git commit -m "feat: add RDP types and API function"
```

---

## Task 6: Frontend ConnectPage with Tabs

**Files:**
- Modify: `frontend/src/pages/ConnectPage.vue`

- [ ] **Step 1: Add tab switching for Linux/Windows**

Read `frontend/src/pages/ConnectPage.vue`. Restructure the form to add tabs:

Replace the existing form template with:

```vue
<template>
  <div class="connect-page">
    <nav class="navbar">
      <div class="brand">GWebSSH</div>
      <div class="nav-links">
        <router-link to="/audit">审计</router-link>
      </div>
    </nav>
    <div class="connect-card">
      <el-tabs v-model="activeTab" class="connect-tabs">
        <el-tab-pane label="Linux (SSH)" name="ssh">
          <el-form :model="sshForm" label-width="80px">
            <el-form-item label="目标IP">
              <el-input v-model="sshForm.target" placeholder="请输入目标IP" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="sshForm.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="sshForm.username" placeholder="请输入用户名" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="sshForm.password" type="password" placeholder="请输入密码" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="connectSSH" :loading="loading">连接</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
        <el-tab-pane label="Windows (RDP)" name="rdp">
          <el-form :model="rdpForm" label-width="80px">
            <el-form-item label="目标IP">
              <el-input v-model="rdpForm.target" placeholder="请输入目标IP" />
            </el-form-item>
            <el-form-item label="端口">
              <el-input-number v-model="rdpForm.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="用户名">
              <el-input v-model="rdpForm.username" placeholder="请输入用户名" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="rdpForm.password" type="password" placeholder="请输入密码" show-password />
            </el-form-item>
            <el-form-item label="域名">
              <el-input v-model="rdpForm.domain" placeholder="可选，如 WORKGROUP" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="connectRDP" :loading="loading">连接</el-button>
            </el-form-item>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>
```

Update the script section:

```vue
<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { obtainKey, obtainKeyRDP } from '../api'

const activeTab = ref('ssh')
const loading = ref(false)

const sshForm = reactive({
  target: '',
  port: 22,
  username: '',
  password: '',
})

const rdpForm = reactive({
  target: '',
  port: 3389,
  username: '',
  password: '',
  domain: '',
})

async function connectSSH() {
  loading.value = true
  try {
    const { key } = await obtainKey(sshForm)
    window.open(`/term?key=${key}&host=${sshForm.target}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e.message || '连接失败')
  } finally {
    loading.value = false
  }
}

async function connectRDP() {
  loading.value = true
  try {
    const { key } = await obtainKeyRDP(rdpForm)
    window.open(`/rdp?key=${key}&host=${rdpForm.target}`, '_blank')
  } catch (e: any) {
    ElMessage.error(e.message || '连接失败')
  } finally {
    loading.value = false
  }
}
</script>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/ConnectPage.vue
git commit -m "feat: add Linux/Windows tab switching to ConnectPage"
```

---

## Task 7: Frontend RDP WebSocket Composable

**Files:**
- Create: `frontend/src/composables/useRdpWebSocket.ts`

- [ ] **Step 1: Create useRdpWebSocket composable**

Create `frontend/src/composables/useRdpWebSocket.ts`:

```typescript
import { ref, onUnmounted } from 'vue'

export type RdpStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export function useRdpWebSocket(key: string) {
  const status = ref<RdpStatus>('connecting')
  const error = ref('')
  let socket: WebSocket | null = null

  function connect(host: string): WebSocket {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    socket = new WebSocket(`${protocol}//${host}/ws/v1/rdp/${key}`)

    socket.addEventListener('open', () => {
      status.value = 'connected'
    })

    socket.addEventListener('close', () => {
      status.value = 'disconnected'
    })

    socket.addEventListener('error', (e) => {
      status.value = 'error'
      error.value = 'WebSocket连接失败'
    })

    return socket
  }

  function getSocket() {
    return socket
  }

  function close() {
    socket?.close()
  }

  onUnmounted(close)

  return { status, error, connect, getSocket, close }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/composables/useRdpWebSocket.ts
git commit -m "feat: add RDP WebSocket composable"
```

---

## Task 8: Frontend RdpPage

**Files:**
- Create: `frontend/src/pages/RdpPage.vue`

- [ ] **Step 1: Create RdpPage with guacamole-common-js**

First, install guacamole-common-js:

```bash
cd frontend && npm install guacamole-common-js
```

Create `frontend/src/pages/RdpPage.vue`:

```vue
<template>
  <div class="rdp-page" ref="rdpContainer">
    <div class="rdp-toolbar">
      <span class="connection-status" :style="{ color: statusColor }">
        ● {{ statusText }}
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

const { status, error, connect, getSocket } = useRdpWebSocket(key)

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
    // Create Guacamole client
    const display = new Guacamole.Display(displayContainer.value!)
    guacClient = new Guacamole.Client(socket as any)

    // Attach display
    guacClient.attach()

    // Get display element and add mouse/keyboard input
    const element = guacClient.getDisplay().getElement()

    // Mouse
    const mouse = new Guacamole.Mouse(element)
    mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState: any) => {
      guacClient.sendMouseState(mouseState)
    }

    // Keyboard
    const keyboard = new Guacamole.Keyboard(document)
    keyboard.onkeydown = (keysym: number) => {
      guacClient.sendKeyEvent(1, keysym)
    }
    keyboard.onkeyup = (keysym: number) => {
      guacClient.sendKeyEvent(0, keysym)
    }

    // Send initial size
    socket.send(JSON.stringify({
      resize: [displayContainer.value?.clientWidth || 1024, displayContainer.value?.clientHeight || 768]
    }))

    // Handle incoming instructions from server
    socket.addEventListener('message', (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.op && guacClient) {
          // Apply instruction to display
          guacClient.getDisplay().eval(msg.op, msg.args || [])
        }
      } catch (e) {
        // Binary or non-JSON data
      }
    })

    // Window resize
    window.addEventListener('resize', () => {
      if (displayContainer.value) {
        socket.send(JSON.stringify({
          resize: [displayContainer.value.clientWidth, displayContainer.value.clientHeight]
        }))
      }
    })
  })
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
```

Note: The guacamole-common-js integration may need adjustments based on the actual API. The `Guacamole.Client` constructor expects a WebSocket-like object, and the instruction handling might differ. This is a starting point that will need testing against the actual guacamole-common-js library.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/RdpPage.vue frontend/package.json frontend/package-lock.json
git commit -m "feat: add RDP page with guacamole-common-js rendering"
```

---

## Task 9: Frontend Router + Cleanup

**Files:**
- Modify: `frontend/src/router/index.ts`

- [ ] **Step 1: Add /rdp route**

Read `frontend/src/router/index.ts`. Add route:

```typescript
{
  path: '/rdp',
  name: 'Rdp',
  component: () => import('../pages/RdpPage.vue'),
}
```

- [ ] **Step 2: Final verification**

Run: `cd frontend && npm run build`
Expected: no errors

Run: `cd backend && go build ./...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add frontend/src/router/index.ts
git commit -m "feat: add /rdp route for RDP remote desktop"
```

---

## Task 10: Integration Testing

- [ ] **Step 1: Start guacd**

```bash
docker compose up -d guacd
```

Verify guacd is running: `docker compose ps guacd`

- [ ] **Step 2: Build and start backend**

```bash
cd backend && go build -o gwebssh ./cmd/ && ./gwebssh
```

- [ ] **Step 3: Build and start frontend**

```bash
cd frontend && npm run dev
```

- [ ] **Step 4: Test RDP connection**

1. Open browser to `http://localhost:5173/connect`
2. Click "Windows (RDP)" tab
3. Fill in RDP credentials (target IP, port 3389, username, password)
4. Click "连接"
5. New tab opens at `/rdp?key=xxx`
6. RDP desktop should appear in browser
7. Mouse and keyboard should work
8. Close the tab — verify .guac file in MinIO and audit record in ES

- [ ] **Step 5: Fix any issues found during testing**

---

## Verification Checklist

- [ ] `docker compose up -d guacd` — guacd starts successfully
- [ ] `cd backend && go build ./...` — compiles without errors
- [ ] `cd frontend && npm run build` — builds without errors
- [ ] ConnectPage shows Linux/Windows tabs
- [ ] Windows tab has RDP form (target, port, username, password, domain)
- [ ] Clicking connect generates key and opens `/rdp?key=xxx`
- [ ] RdpPage connects to WebSocket and renders RDP desktop
- [ ] Mouse movements are reflected on remote Windows
- [ ] Keyboard input works
- [ ] Disconnecting creates .guac file in MinIO
- [ ] ES has RDP login audit record
