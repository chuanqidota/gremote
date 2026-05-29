import axios from 'axios'
import type { SSHInfo, RDPInfo, FileItem, AuditRecord, AuditQuery } from '../types'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

export async function obtainKey(info: SSHInfo): Promise<string> {
  const { data } = await http.post('/obtain-key', info)
  return data.key
}

export async function obtainKeyRDP(info: RDPInfo): Promise<string> {
  const { data } = await http.post('/obtain-key-rdp', info)
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

export function getRecordFileGuacUrl(key: string): string {
  return `/api/v1/record-file-guac?key=${encodeURIComponent(key)}`
}

export function getRecordFileMP4Url(key: string): string {
  return `/api/v1/record-file-mp4?key=${encodeURIComponent(key)}`
}

export async function getConvertStatus(key: string): Promise<{ converted: boolean; converting?: boolean; mp4_url?: string; step?: string; progress?: number; error?: string }> {
  const { data } = await http.get('/convert-status', { params: { key } })
  return data.data
}

export async function triggerConvert(key: string): Promise<void> {
  await http.post('/convert-guac', null, { params: { key } })
}

export async function getConfig(): Promise<{ display_mode: string }> {
  const { data } = await http.get('/config')
  return data.data
}
