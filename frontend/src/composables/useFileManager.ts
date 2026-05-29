import { ref } from 'vue'
import { listFiles, uploadFile, getDownloadUrl } from '../api'
import { extractErrorMessage } from '../utils/error'
import type { FileItem } from '../types'

export function useFileManager(key: string) {
  const files = ref<FileItem[]>([])
  const loading = ref(false)
  const error = ref('')
  const currentPath = ref('/tmp')

  async function fetchFiles(path: string) {
    loading.value = true
    error.value = ''
    try {
      files.value = await listFiles(key, path)
      currentPath.value = path
    } catch (e: any) {
      error.value = extractErrorMessage(e, 'Failed to list files')
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
      error.value = extractErrorMessage(e, 'Upload failed')
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
