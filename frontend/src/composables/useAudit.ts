import { ref } from 'vue'
import { queryAudit, getRecordUrl, getRecordFileGuacUrl, getConvertStatus, triggerConvert, getRecordFileMP4Url } from '../api'
import { extractErrorMessage } from '../utils/error'
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
      error.value = extractErrorMessage(e, 'Failed to fetch audit records')
    } finally {
      loading.value = false
    }
  }

  async function fetchRecordUrl(key: string): Promise<string> {
    return getRecordUrl(key)
  }

  function fetchGuacRecordUrl(key: string): string {
    return getRecordFileGuacUrl(key)
  }

  function fetchConvertStatus(key: string) {
    return getConvertStatus(key)
  }

  function fetchTriggerConvert(key: string) {
    return triggerConvert(key)
  }

  function fetchRecordFileMP4Url(key: string): string {
    return getRecordFileMP4Url(key)
  }

  return { data, count, loading, error, fetch, fetchRecordUrl, fetchGuacRecordUrl, fetchConvertStatus, fetchTriggerConvert, fetchRecordFileMP4Url }
}
