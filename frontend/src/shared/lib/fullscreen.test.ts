import { createRoot } from 'solid-js'
import { describe, expect, it } from 'vitest'
import { createFullscreen } from './fullscreen'

type Listener = () => void

/** A Document stand-in carrying only the Fullscreen surface the wrapper reads. */
class FakeFullscreenDocument {
  fullscreenElement: Element | null = null
  readonly documentElement: { requestFullscreen: () => Promise<void> }
  private readonly listeners = new Map<string, Set<Listener>>()

  constructor(readonly fullscreenEnabled = true) {
    this.documentElement = {
      requestFullscreen: () => {
        this.fullscreenElement = this.documentElement as unknown as Element
        this.emit('fullscreenchange')
        return Promise.resolve()
      },
    }
  }

  exitFullscreen = () => {
    this.fullscreenElement = null
    this.emit('fullscreenchange')
    return Promise.resolve()
  }

  addEventListener(type: string, listener: Listener) {
    const set = this.listeners.get(type) ?? new Set<Listener>()
    set.add(listener)
    this.listeners.set(type, set)
  }

  removeEventListener(type: string, listener: Listener) {
    this.listeners.get(type)?.delete(listener)
  }

  emit(type: string) {
    for (const listener of this.listeners.get(type) ?? []) listener()
  }

  listenerCount(type: string) {
    return this.listeners.get(type)?.size ?? 0
  }

  asDocument() {
    return this as unknown as Document
  }
}

/** An older iPad / pre-16.4 Safari: only the prefixed names exist. */
class FakeWebkitFullscreenDocument {
  webkitFullscreenElement: Element | null = null
  webkitFullscreenEnabled = true
  webkitRequests = 0
  readonly documentElement: { webkitRequestFullscreen: () => void }

  constructor() {
    this.documentElement = {
      webkitRequestFullscreen: () => {
        this.webkitRequests += 1
      },
    }
  }

  addEventListener() {}
  removeEventListener() {}

  asDocument() {
    return this as unknown as Document
  }
}

describe('createFullscreen', () => {
  it('entra e sai da tela cheia pelo mesmo toggle', () => {
    const doc = new FakeFullscreenDocument()
    createRoot((dispose) => {
      const fullscreen = createFullscreen(doc.asDocument())
      expect(fullscreen.active()).toBe(false)

      fullscreen.toggle()
      expect(fullscreen.active()).toBe(true)

      fullscreen.toggle()
      expect(fullscreen.active()).toBe(false)
      dispose()
    })
  })

  // O jogador também sai pelo Esc e pelo gesto do sistema: o rótulo tem de
  // seguir o browser, não a intenção do último clique.
  it('acompanha uma saída que não veio do toggle', () => {
    const doc = new FakeFullscreenDocument()
    createRoot((dispose) => {
      const fullscreen = createFullscreen(doc.asDocument())
      fullscreen.toggle()
      expect(fullscreen.active()).toBe(true)

      doc.fullscreenElement = null
      doc.emit('fullscreenchange')
      expect(fullscreen.active()).toBe(false)
      dispose()
    })
  })

  // `supported` é o que decide se o item existe no menu — no iPhone ele some.
  it('reporta não-suportado quando o browser não tem a API', () => {
    const doc = new FakeFullscreenDocument(false)
    createRoot((dispose) => {
      expect(createFullscreen(doc.asDocument()).supported).toBe(false)
      dispose()
    })
  })

  it('usa os nomes webkit quando são os únicos que existem', () => {
    const doc = new FakeWebkitFullscreenDocument()
    createRoot((dispose) => {
      const fullscreen = createFullscreen(doc.asDocument())
      expect(fullscreen.supported).toBe(true)

      fullscreen.toggle()
      expect(doc.webkitRequests).toBe(1)
      dispose()
    })
  })

  it('solta os listeners ao descartar o escopo', () => {
    const doc = new FakeFullscreenDocument()
    createRoot((dispose) => {
      createFullscreen(doc.asDocument())
      expect(doc.listenerCount('fullscreenchange')).toBe(1)
      dispose()
    })
    expect(doc.listenerCount('fullscreenchange')).toBe(0)
  })
})
