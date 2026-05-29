import { computed, type Ref } from 'vue'

const defaultTextMap: Record<string, string> = {
  connecting: '连接中...',
  connected: '已连接',
  disconnected: '已断开',
}

const colorMap: Record<string, string> = {
  connected: '#67c23a',
  connecting: '#e6a23c',
  error: '#f56c6c',
}

export function useConnectionStatus(
  status: Ref<string>,
  error: Ref<string>,
  textMap?: Record<string, string>,
) {
  const mergedTextMap = { ...defaultTextMap, ...textMap }

  const statusColor = computed(() => colorMap[status.value] || '#909399')

  const statusText = computed(() => {
    if (status.value === 'error') return error.value || '错误'
    return mergedTextMap[status.value] || ''
  })

  return { statusColor, statusText }
}
