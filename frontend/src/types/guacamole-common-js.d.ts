declare module 'guacamole-common-js' {
  export class Display {
    getElement(): HTMLDivElement
    eval(opcode: string, args: string[]): void
  }

  export class Client {
    constructor(tunnel: any)
    getDisplay(): Display
    connect(): void
    disconnect(): void
    sendMouseState(state: any): void
    sendKeyEvent(pressed: number, keysym: number): void
    attach(): void
  }

  export class WebSocketTunnel {
    constructor(socket: WebSocket)
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
