declare module 'element-plus/dist/locale/zh-cn.mjs'

declare module 'asciinema-player' {
  export function create(
    src: string | { driver?: string; data?: string | Blob; url?: string },
    element: HTMLElement,
    opts?: {
      autoPlay?: boolean
      speed?: number
      idleTimeLimit?: number
      cols?: number
      rows?: number
      theme?: string
      preload?: boolean
      loop?: boolean
      startAt?: number
      poster?: string
      fit?: string
      fontSize?: string
    }
  ): void
}
