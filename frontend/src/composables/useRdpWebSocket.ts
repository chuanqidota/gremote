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

    socket.addEventListener('error', () => {
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
