import { ref } from 'vue'
import { queryAudit, getRecordUrl, getRecordFileGuacUrl, getConvertStatus, triggerConvert, getRecordFileMP4Url } from '../api'
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

  function fetchGuacRecordUrl(key: string): string {
    return getRecordFileGuacUrl(key)
  }

  return { data, count, loading, error, fetch, fetchRecordUrl, fetchGuacRecordUrl, getConvertStatus, triggerConvert, getRecordFileMP4Url }
}
