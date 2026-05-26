declare module 'guacamole-common-js' {
  export class Display {
    constructor()
    cursorHotspotX: number
    cursorHotspotY: number
    cursorX: number
    cursorY: number
    getElement(): HTMLDivElement
    getDisplayElement(): HTMLDivElement
    getDefaultLayer(): VisibleLayer
    getCursorLayer(): VisibleLayer
    createLayer(): VisibleLayer
    createBuffer(): Layer
    getWidth(): number
    getHeight(): number
    resize(layer: any, width: number, height: number): void
    draw(layer: any, x: number, y: number, url: string): void
    drawBlob(layer: any, x: number, y: number, blob: Blob): void
    drawImage(layer: any, x: number, y: number, image: CanvasImageSource): void
    drawStream(layer: any, x: number, y: number, stream: any, mimetype: string): void
    rect(layer: any, x: number, y: number, w: number, h: number): void
    clip(layer: any): void
    setChannelMask(layer: any, mask: number): void
    fillColor(layer: any, r: number, g: number, b: number, a: number): void
    strokeColor(layer: any, cap: string, join: string, thickness: number, r: number, g: number, b: number, a: number): void
    setCursor(hotspotX: number, hotspotY: number, srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number): void
    showCursor(shown: boolean): void
    moveCursor(x: number, y: number): void
    moveTo(layer: any, x: number, y: number): void
    lineTo(layer: any, x: number, y: number): void
    close(layer: any): void
    push(layer: any): void
    pop(layer: any): void
    reset(layer: any): void
    setTransform(layer: any, a: number, b: number, c: number, d: number, e: number, f: number): void
    transform(layer: any, a: number, b: number, c: number, d: number, e: number, f: number): void
    copy(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, dstLayer: any, x: number, y: number): void
    put(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, dstLayer: any, x: number, y: number): void
    move(layer: any, parent: any, x: number, y: number, z: number): void
    dispose(layer: any): void
    shade(layer: any, alpha: number): void
    distort(layer: any, a: number, b: number, c: number, d: number, e: number, f: number): void
    scale(scale: number): void
    onresize: ((width: number, height: number) => void) | null
    flush(callback?: () => void, timestamp?: number, logicalFrames?: number): void
    cancel(): void
    flatten(): HTMLCanvasElement
  }

  export class VisibleLayer {
    width: number
    height: number
    x: number
    y: number
    z: number
    alpha: number
    getElement(): HTMLDivElement
    getCanvas(): HTMLCanvasElement
    toCanvas(): HTMLCanvasElement
    resize(width: number, height: number): void
    drawImage(x: number, y: number, image: CanvasImageSource): void
    moveTo(x: number, y: number): void
    lineTo(x: number, y: number): void
    close(): void
    rect(x: number, y: number, w: number, h: number): void
    clip(): void
    push(): void
    pop(): void
    reset(): void
    fillColor(r: number, g: number, b: number, a: number): void
    strokeColor(cap: string, join: string, thickness: number, r: number, g: number, b: number, a: number): void
    setChannelMask(mask: number): void
    setTransform(a: number, b: number, c: number, d: number, e: number, f: number): void
    transform(a: number, b: number, c: number, d: number, e: number, f: number): void
    translate(x: number, y: number): void
    move(parent: VisibleLayer | Layer, x: number, y: number, z: number): void
    dispose(): void
    shade(alpha: number): void
    distort(a: number, b: number, c: number, d: number, e: number, f: number): void
    copy(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number): void
    put(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number): void
    transfer(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number, transferFunction: any): void
    fillLayer(srcLayer: any): void
    strokeLayer(cap: string, join: string, thickness: number, srcLayer: any): void
  }

  export class Layer {
    width: number
    height: number
    autosize: boolean
    getCanvas(): HTMLCanvasElement
    toCanvas(): HTMLCanvasElement
    resize(width: number, height: number): void
    drawImage(x: number, y: number, image: CanvasImageSource): void
    moveTo(x: number, y: number): void
    lineTo(x: number, y: number): void
    close(): void
    rect(x: number, y: number, w: number, h: number): void
    clip(): void
    push(): void
    pop(): void
    reset(): void
    fillColor(r: number, g: number, b: number, a: number): void
    strokeColor(cap: string, join: string, thickness: number, r: number, g: number, b: number, a: number): void
    setChannelMask(mask: number): void
    setTransform(a: number, b: number, c: number, d: number, e: number, f: number): void
    transform(a: number, b: number, c: number, d: number, e: number, f: number): void
    copy(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number): void
    put(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number): void
    transfer(srcLayer: any, srcx: number, srcy: number, srcw: number, srch: number, x: number, y: number, transferFunction: any): void
    fillLayer(srcLayer: any): void
    strokeLayer(cap: string, join: string, thickness: number, srcLayer: any): void
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
    oninstruction: ((opcode: string, args: string[]) => void) | null
    connect(data?: string): void
    disconnect(): void
    sendMessage(...elements: any[]): void
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
    seek(position: number, callback: () => void): void
    isPlaying(): boolean
    cancel(): void
    getState(): number
    getDuration(): number
    getPosition(): number
    getDisplay(): Display
    oninstruction: ((opcode: string, args: string[]) => void) | null
    onprogress: ((duration: number, parsedSize: number) => void) | null
    ondurationchange: ((duration: number) => void) | null
    onplay: (() => void) | null
    onpause: (() => void) | null
    onseek: ((position: number, current: number, total: number) => void) | null
    onerror: ((errorMsg: any) => void) | null
    onload: (() => void) | null
  }

  export class Parser {
    oninstruction: ((opcode: string, args: string[]) => void) | null
    receive(packet: string): void
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
