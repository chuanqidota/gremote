declare module 'guacamole-common-js' {
  export class Display {
    getElement(): HTMLDivElement
    eval(opcode: string, args: string[]): void
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
