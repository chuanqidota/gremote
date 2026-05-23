declare module 'asciinema-player' {
  export function create(
    src: string,
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
