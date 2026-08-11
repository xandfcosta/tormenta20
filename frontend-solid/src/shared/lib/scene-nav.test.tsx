import { render } from '@solidjs/testing-library'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSceneNav } from './scene-nav'

// jsdom has no layout — feed each node a fake rect so the geometry core runs.
function stubRect(el: Element, l: number, t: number, r: number, b: number): void {
  ;(el as HTMLElement).getBoundingClientRect = () =>
    ({
      left: l,
      top: t,
      right: r,
      bottom: b,
      width: r - l,
      height: b - t,
      x: l,
      y: t,
      toJSON: () => ({}),
    }) as DOMRect
}

function mockDesktop(matches = true): void {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

function press(key: string): void {
  window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }))
}

const sfx = vi.fn()
const onEscape = vi.fn()
const bumpers = { prev: vi.fn(), next: vi.fn() }

function Scene() {
  let root: HTMLDivElement | undefined
  createSceneNav({ root: () => root ?? null, onEscape, bumpers, sfx })
  return (
    <div data-tome-root ref={root}>
      {/* `data-selected` is Kobalte's active-tab marker (Radix used data-state). */}
      <div data-nav-region="rail" data-nav-layout="column" role="tablist">
        <button type="button" role="tab" data-selected>
          S1
        </button>
        <button type="button" role="tab">
          S2
        </button>
      </div>
      <div data-nav-region="content" data-nav-layout="grid" data-nav-edge-left="rail">
        <a href="#a">A</a>
        <a href="#b">B</a>
        <a href="#c">C</a>
        <a href="#d">D</a>
        <input aria-label="busca" />
      </div>
    </div>
  )
}

function setup() {
  const view = render(() => <Scene />)
  const g = (t: string) => view.getByText(t)
  stubRect(view.container.querySelector('[data-nav-region="rail"]')!, 0, 60, 80, 150)
  stubRect(view.container.querySelector('[data-nav-region="content"]')!, 120, 60, 300, 150)
  stubRect(g('S1'), 0, 60, 80, 100)
  stubRect(g('S2'), 0, 110, 80, 150)
  stubRect(g('A'), 120, 60, 200, 100)
  stubRect(g('B'), 220, 60, 300, 100)
  stubRect(g('C'), 120, 110, 200, 150)
  stubRect(g('D'), 220, 110, 300, 150)
  stubRect(view.getByLabelText('busca'), 120, 160, 300, 190)
  return { view, g }
}

beforeEach(() => {
  mockDesktop(true)
  Element.prototype.scrollIntoView = vi.fn()
})
afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('createSceneNav', () => {
  it('anda dentro de um grid por geometria (→ e ↓)', () => {
    const { g } = setup()
    g('A').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('B'))
    press('ArrowDown')
    expect(document.activeElement).toBe(g('D'))
  })

  it('cruza do conteúdo pro rail na borda esquerda', () => {
    const { g } = setup()
    g('A').focus()
    press('ArrowLeft')
    expect(document.activeElement).toBe(g('S1')) // aba ativa do rail
  })

  it('mergulha do rail pro conteúdo no →', () => {
    const { g } = setup()
    g('S1').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('A'))
  })

  it('Esc no conteúdo volta pro rail (sem sair da cena)', () => {
    const { g } = setup()
    g('A').focus()
    press('Escape')
    expect(document.activeElement).toBe(g('S1'))
    expect(onEscape).not.toHaveBeenCalled()
  })

  it('Esc no rail sai da cena', () => {
    const { g } = setup()
    g('S1').focus()
    press('Escape')
    expect(onEscape).toHaveBeenCalledOnce()
  })

  it('PageDown / PageUp disparam os bumpers de seção', () => {
    const { g } = setup()
    g('A').focus()
    press('PageDown')
    press('PageUp')
    expect(bumpers.next).toHaveBeenCalledOnce()
    expect(bumpers.prev).toHaveBeenCalledOnce()
  })

  // "Ponta" é ordem de DOM, não geometria. E o input de busca NÃO conta: a
  // lista FOCUSABLE cobre a/button/[tabindex]/[data-nav-item], de propósito —
  // as setas não devem parar num campo de texto.
  it('Home/End vão pras pontas da região, ignorando campos de texto', () => {
    const { g } = setup()
    g('B').focus()
    press('Home')
    expect(document.activeElement).toBe(g('A'))
    press('End')
    expect(document.activeElement).toBe(g('D'))
  })

  it('fica quieto enquanto o usuário digita num input', () => {
    const { view } = setup()
    const input = view.getByLabelText('busca')
    input.focus()
    press('ArrowLeft')
    expect(document.activeElement).toBe(input) // inalterado
  })

  it('não faz nada quando o gate de mídia está fechado', () => {
    vi.clearAllMocks()
    mockDesktop(false)
    const { g } = setup()
    g('A').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('A')) // nenhum listener ligado
  })

  it('lembra o cursor ao reentrar numa região', () => {
    const { g } = setup()
    g('A').focus()
    press('ArrowRight') // cursor do conteúdo agora em B
    press('ArrowLeft') // B → borda? não: volta pra A
    expect(document.activeElement).toBe(g('A'))
    press('ArrowLeft') // agora sim cruza pro rail
    expect(document.activeElement).toBe(g('S1'))
    press('ArrowRight') // reentra no conteúdo: cursor lembrado
    expect(document.activeElement).toBe(g('A'))
  })
})

describe('createSceneNav — delegado (cenas por seleção)', () => {
  const onCommand = vi.fn<(c: unknown) => boolean>(() => true)
  const onKey = vi.fn<(e: KeyboardEvent) => boolean>(() => false)
  const leaveScene = vi.fn()

  function Delegated() {
    let root: HTMLDivElement | undefined
    createSceneNav({
      root: () => root ?? null,
      delegated: true,
      onCommand,
      onKey,
      onEscape: leaveScene,
      sfx,
    })
    return <div data-slot="scene-shell" ref={root} />
  }

  it('mapeia a gramática padrão pro onCommand', () => {
    render(() => <Delegated />)
    press('ArrowRight')
    press('Enter')
    press('PageDown')
    expect(onCommand).toHaveBeenCalledWith({ type: 'move', dir: 'right' })
    expect(onCommand).toHaveBeenCalledWith({ type: 'activate' })
    expect(onCommand).toHaveBeenCalledWith({ type: 'bumper', dir: 'next' })
  })

  it('roteia teclas próprias pelo onKey (mesmo com input focado)', () => {
    onKey.mockImplementation((e) => e.key === 'd')
    render(() => <Delegated />)
    press('d')
    expect(onKey).toHaveBeenCalled()
  })

  it('cai no onEscape quando o onCommand não trata o Esc', () => {
    onCommand.mockImplementation((c) => (c as { type: string }).type !== 'back')
    render(() => <Delegated />)
    press('Escape')
    expect(leaveScene).toHaveBeenCalledOnce()
  })
})
