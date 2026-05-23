export interface SSHInfo {
  target: string
  username: string
  password: string
  port: number
  user?: string
  source?: string
}

export interface FileItem {
  name: string
  size: number
  type: 'file' | 'directory'
}

export interface AuditRecord {
  key: string
  user: string
  source: string
  target: string
  startTime: string
  endTime: string
}

export interface AuditQuery {
  offset: number
  limit: number
  user?: string
  source?: string
  target?: string
  startTime?: string
  endTime?: string
  search?: string
}
