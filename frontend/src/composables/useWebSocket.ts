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
