# WebSSH Frontend Design

## Overview

Rebuild the webssh frontend using Vue 3 + Element Plus + TypeScript + Vite. The project splits into two areas: a standalone terminal page (with integrated file manager) that external systems can link to directly, and management pages for connections, audit logs, and session playback.

## Routes

| Path | Page | Description |
|---|---|---|
| `/connect` | ConnectPage | SSH connection form |
| `/term?key=xxx` | TerminalPage | xterm terminal + file toolbar (standalone, key-driven) |
| `/audit` | AuditPage | Login audit table with search/filter/pagination |
| `/playback?key=xxx` | PlaybackPage | Asciinema session playback |

## Architecture

```
frontend/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router/index.ts
│   ├── api/index.ts            # Axios instance, typed API functions
│   ├── types/index.ts          # TypeScript interfaces
│   ├── composables/
│   │   ├── useWebSocket.ts     # WS connect/send/receive/close lifecycle
│   │   ├── useFileManager.ts   # List/upload/download via SFTP APIs
│   │   └── useAudit.ts         # Audit query + record URL
│   ├── pages/
│   │   ├── ConnectPage.vue     # SSH form → obtain-key → open terminal
│   │   ├── TerminalPage.vue    # xterm + file toolbar + dialogs
│   │   ├── AuditPage.vue       # Login audit table
│   │   └── PlaybackPage.vue    # Asciinema player
│   └── components/
│       ├── FileToolbar.vue     # Upload/download/browse buttons
│       ├── FileListDialog.vue  # SFTP file browser dialog
│       └── FileUploadDialog.vue # Upload dialog
```

## Data Flow

1. ConnectPage: user fills SSH form → `POST /ws/v1/obtain-key` → opens `/term?key=xxx` in new tab
2. TerminalPage: reads `key` from route query → opens `ws://host/ws/v1/{key}` → xterm bridges I/O
3. File operations: use `key` + path → `GET/POST` list-file/upload-file/download-file
4. AuditPage: `GET /ws/v1/login-audit` with offset/limit/search/filters
5. PlaybackPage: `GET /ws/v1/record-url?key=xxx` → loads asciinema player with returned URL

## Page Details

### ConnectPage
- El-form: host, port (default 22), username, password
- Validation via Element Plus form rules
- On success: `window.open('/term?key=' + key, '_blank')`
- Loading state on submit button

### TerminalPage
- Reads `key` from `route.query.key`
- `useWebSocket` composable: manages WS lifecycle, exposes `{ status, error, send }`
- On WS open: creates xterm instance with FitAddon, sends initial resize
- Auto-fit on window resize, forwards resize events to backend
- Toolbar above terminal: Upload / Download / Browse Files buttons
- FileListDialog: table with name/size/type, navigation, click to download or select path for upload
- FileUploadDialog: file picker + target path
- All file dialogs receive `key` as prop

### AuditPage
- El-table with columns: user, source, target, startTime, endTime, operations
- Pagination (offset/limit)
- Search bar + optional date range filter
- Playback button in operations column: `window.open('/playback?key=' + row.key, '_blank')`

### PlaybackPage
- Reads `key` from route query
- Calls `GET /ws/v1/record-url?key=xxx`
- Shows loading spinner while fetching
- Renders asciinema player with autoPlay

## Composables

### useWebSocket(key: string)
- Returns: `{ status, error, termRef, send }`
- Manages: connect, reconnect not supported (keys are one-time use)
- Handles: resize sync, binary/text message routing

### useFileManager(key: string)
- Returns: `{ listFiles, uploadFile, downloadFile, loading, error }`
- Wraps the three SFTP API endpoints

### useAudit()
- Returns: `{ data, count, loading, error, fetch, search, paginate }`
- Manages audit query state

## Backend Fixes

- Fix `params.go:8`: `json:""` → `json:"target"` on Target field
- Fix `recordAudit/record.go:63`: parameterize ES query instead of `fmt.Sprintf` with raw key
- Fix `redis/redis.go:80`: `IsConnected` race condition — use `SetNX` (atomic set-if-not-exists)
- Fix route naming: move REST routes out of `/ws` group into an `/api` group, keep only WebSocket under `/ws`

## Dependencies

- vue 3, vue-router 4
- element-plus
- xterm, xterm-addon-fit, xterm-addon-attach
- asciinema-player
- axios
- typescript, vite, @vitejs/plugin-vue

## Non-Goals

- Authentication system (internal tool)
- Dark mode
- i18n
- Mobile responsive (desktop tool)
- Session persistence / reconnection
