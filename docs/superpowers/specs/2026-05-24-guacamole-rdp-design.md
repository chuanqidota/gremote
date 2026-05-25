# Guacamole RDP Integration Design

## Context

GWebSSH 项目当前仅支持 SSH 连接。需要新增 Windows RDP 远程桌面支持，使用 Apache Guacamole 生态（guacd + guacamole-common-js），实现 RDP 连接和会话录像。

## Architecture

```
浏览器 ──WebSocket──> Go 后端 ──Guacamole协议(TCP)──> guacd ──RDP──> Windows
                         │
                    Redis (key)
                    ES (录像元数据)
                    MinIO (.guac文件)
```

## Components

### 1. guacd (Docker)

- 官方镜像 `guacamole/guacd`
- 监听 TCP 4822 端口
- 处理 RDP 协议，原生支持录制为 .guac 格式
- Go 后端通过 Guacamole 协议与 guacd 通信

### 2. Go 后端

**新增文件：**
- `pkg/guacamole/client.go` — Guacamole 协议客户端（解析/生成指令）
- `pkg/guacamole/protocol.go` — 协议指令定义
- `app/ws/view/rdp.go` — RDP WebSocket 处理器
- `app/api/view/view.go` — 新增 `ObtainRdpKey` 接口

**复用现有组件：**
- Redis key 存储（和 SSH 共用）
- ES 审计写入（扩展支持 rdp 类型）
- MinIO 存储（存 .guac 文件）

### 3. 前端

**修改：**
- `ConnectPage.vue` — 添加 Linux/Windows tab 切换
  - Linux tab：现有 SSH 表单（目标IP、端口、用户名、密码）
  - Windows tab：RDP 表单（目标IP、端口、用户名、密码、域名）

**新增：**
- `RdpPage.vue` — RDP 远程桌面页面
- `composables/useGuacamole.ts` — guacamole-common-js 封装

**路由：**
- `/rdp?key=<key>` — RDP 远程桌面页面

## Data Flow

### 连接流程

1. 用户在 ConnectPage 选择 Windows tab，填写 RDP 凭据
2. 前端调用 `POST /api/v1/obtain-rdp-key`，后端生成 UUID key 存入 Redis
3. 前端跳转到 `/rdp?key=<key>`
4. RdpPage 建立 WebSocket 连接到 `ws://host/ws/v1/rdp/<key>`
5. 后端从 Redis 取出凭据，通过 Guacamole 协议连接 guacd
6. guacd 发起 RDP 连接到目标 Windows 机器
7. 三个 goroutine 桥接数据：
   - `ReceiveWsMsg` — WebSocket 输入 → guacd
   - `WriteWsMsg` — guacd 输出 → WebSocket
   - `WriteRdpData` — guacd 输出 → MinIO (.guac 录像)

### 录像流程

1. guacd 在处理 RDP 数据流时，Go 后端将所有 guacd 指令同时写入 MinIO
2. 录像元数据（key、用户、目标、时间）写入 ES，类型标记为 `rdp`
3. 回放页面支持 .guac 格式播放

## Guacamole Protocol

Go 后端与 guacd 通信使用 Guacamole 协议：

### 指令格式

```
length.field1,length.field2,...,length.fieldN;
```

每条指令以分号结尾，字段长度前缀用逗号分隔。

### 关键指令

**初始化：**
- `select` — 选择协议（`rdp`）
- `hostname` — 目标主机
- `port` — 端口
- `username` — 用户名
- `password` — 密码
- `domain` — 域（可选）
- `security` — 安全模式（如 `any`、`tls`、`nla`）
- `ignore-cert` — 忽略证书

**交互：**
- 客户端 → guacd：`key`（键盘）、`mouse`（鼠标）、`size`（窗口大小）
- guacd → 客户端：`img`（图像）、`rect`（矩形区域）、`copy`（复制）、`fill`（填充）、`sync`（同步）

**录制：**
- 所有 guacd → 客户端的指令流即为 .guac 录像内容

## Docker Changes

`docker-compose.yaml` 新增：

```yaml
guacd:
  image: guacamole/guacd
  restart: always
  ports:
    - "4822:4822"
```

## Config Changes

`config.yaml` 新增：

```yaml
guacd:
  host: guacd
  port: 4822
```

## Verification

1. `docker compose up -d guacd` — guacd 启动成功
2. `cd backend && go build ./...` — 编译通过
3. 前端 ConnectPage 显示 Linux/Windows tab
4. 选择 Windows tab，填写 RDP 凭据，点击连接
5. RdpPage 显示 Windows 桌面画面
6. 操作鼠标键盘，画面实时更新
7. 断开连接后，MinIO 中有 .guac 录像文件
8. ES 中有 RDP 类型的审计记录
9. 回放页面可以播放 .guac 录像
