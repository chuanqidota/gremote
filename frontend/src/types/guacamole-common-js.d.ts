declare module 'guacamole-common-js' {
  export class Display {
    getElement(): HTMLDivElement
    eval(opcode: string, args: string[]): void
    getDisplayElement(): HTMLDivElement
    showCursor(shown: boolean): void
    setCursor(hotspotX: number, hotspotY: number, layer: any, srcx: number, srcy: number, srcw: number, srch: number): void
  }

  export class Client {
    constructor(tunnel: any)
    getDisplay(): Display
    connect(data?: string): void
    disconnect(): void
    sendMouseState(state: any): void
    sendKeyEvent(pressed: number, keysym: number): void
    sendSize(width: number, height: number): void
    attach(): void
    end(): void
  }

  export class Tunnel {
    static State: {
      CLOSED: number
      CONNECTING: number
      OPEN: number
    }
  }

  export class WebSocketTunnel {
    constructor(url: string)
    connect(data?: string): void
    disconnect(): void
    onstatechange: ((state: number) => void) | null
    onerror: ((errorMsg: any) => void) | null
    oninstruction: ((opcode: string, args: string[]) => void) | null
    sendMessage(...elements: any[]): void
  }

  export class StaticHTTPTunnel {
    constructor(url: string, crossDomain?: boolean, extraTunnelHeaders?: Record<string, string>)
    connect(data?: string): void
    disconnect(): void
    onstatechange: ((state: number) => void) | null
    onerror: ((errorMsg: any) => void) | null
    oninstruction: ((opcode: string, args: string[]) => void) | null
    sendMessage(...elements: any[]): void
  }

  export class SessionRecording {
    constructor(source: Blob | Tunnel)
    connect(data?: string): void
    play(): void
    pause(): void
    seek(position: number): void
    getState(): number
    getDuration(): number
    getPosition(): number
    oninstruction: ((opcode: string, args: string[]) => void) | null
    onstatechange: ((state: number) => void) | null
    onprogress: ((position: number) => void) | null
    ondurationchange: ((duration: number) => void) | null
    onplay: (() => void) | null
    onpause: (() => void) | null
    onseek: ((position: number) => void) | null
    onerror: ((errorMsg: any) => void) | null
    static State: {
      CLOSED: number
      CONNECTING: number
      OPEN: number
    }
    static seekMode: {
      RELATIVE: number
      ABSOLUTE: number
    }
  }

  export class Mouse {
    constructor(element: HTMLElement)
    onmousedown: (state: any) => void
    onmouseup: (state: any) => void
    onmousemove: (state: any) => void
  }

  export class Keyboard {
    constructor(target: HTMLElement | Document)
    onkeydown: (keysym: number) => void
    onkeyup: (keysym: number) => void
  }
}
