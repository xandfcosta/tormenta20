import { render } from '@testing-library/react'
import { useRef } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useSceneNav } from './use-scene-nav'

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
  window.dispatchEvent(
    new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true }),
  )
}

const sfx = vi.fn()
const onEscape = vi.fn()
const bumpers = { prev: vi.fn(), next: vi.fn() }

function Scene() {
  const root = useRef<HTMLDivElement>(null)
  useSceneNav({ root: () => root.current, onEscape, bumpers, sfx })
  return (
    <div data-tome-root ref={root}>
      <div data-nav-region="rail" data-nav-layout="column" role="tablist">
        <button type="button" role="tab" data-state="active">
          S1
        </button>
        <button type="button" role="tab" data-state="inactive">
          S2
        </button>
      </div>
      <div
        data-nav-region="content"
        data-nav-layout="grid"
        data-nav-edge-left="rail"
      >
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
  const view = render(<Scene />)
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

describe('useSceneNav', () => {
  it('moves within a grid by geometry (→ and ↓)', () => {
    const { g } = setup()
    g('A').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('B'))
    press('ArrowDown')
    expect(document.activeElement).toBe(g('D'))
  })

  it('crosses from content to the rail at the left edge', () => {
    const { g } = setup()
    g('A').focus()
    press('ArrowLeft')
    expect(document.activeElement).toBe(g('S1')) // rail active tab
  })

  it('dives from the rail into the content on →', () => {
    const { g } = setup()
    g('S1').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('A'))
  })

  it('Esc from content returns to the rail (not leaving the scene)', () => {
    const { g } = setup()
    g('A').focus()
    press('Escape')
    expect(document.activeElement).toBe(g('S1'))
    expect(onEscape).not.toHaveBeenCalled()
  })

  it('Esc from the rail leaves the scene', () => {
    const { g } = setup()
    g('S1').focus()
    press('Escape')
    expect(onEscape).toHaveBeenCalledOnce()
  })

  it('PageDown / PageUp fire the section bumpers', () => {
    const { g } = setup()
    g('A').focus()
    press('PageDown')
    press('PageUp')
    expect(bumpers.next).toHaveBeenCalledOnce()
    expect(bumpers.prev).toHaveBeenCalledOnce()
  })

  it('stands down while typing in an input', () => {
    const { view, g } = setup()
    const input = view.getByLabelText('busca')
    input.focus()
    press('ArrowLeft')
    expect(document.activeElement).toBe(input) // unchanged
    expect(g).toBeTruthy()
  })

  it('does nothing when the media gate is off', () => {
    vi.clearAllMocks()
    mockDesktop(false)
    const { g } = setup()
    g('A').focus()
    press('ArrowRight')
    expect(document.activeElement).toBe(g('A')) // no listener bound
  })
})

describe('useSceneNav — delegated (selection scenes)', () => {
  const onCommand = vi.fn<(c: unknown) => boolean>(() => true)
  const onKey = vi.fn<(e: KeyboardEvent) => boolean>(() => false)
  const leaveScene = vi.fn()

  function Delegated() {
    const root = useRef<HTMLDivElement>(null)
    useSceneNav({
      root: () => root.current,
      delegated: true,
      onCommand,
      onKey,
      onEscape: leaveScene,
      sfx,
    })
    return <div data-slot="scene-shell" ref={root} />
  }

  it('maps the standard grammar to onCommand', () => {
    render(<Delegated />)
    press('ArrowRight')
    press('Enter')
    press('PageDown')
    expect(onCommand).toHaveBeenCalledWith({ type: 'move', dir: 'right' })
    expect(onCommand).toHaveBeenCalledWith({ type: 'activate' })
    expect(onCommand).toHaveBeenCalledWith({ type: 'bumper', dir: 'next' })
  })

  it('routes bespoke keys through onKey (even while an input has focus)', () => {
    onKey.mockImplementation((e) => e.key === 'd')
    render(<Delegated />)
    press('d')
    expect(onKey).toHaveBeenCalled()
  })

  it('falls back to onEscape when onCommand leaves Esc unhandled', () => {
    onCommand.mockImplementation((c) => (c as { type: string }).type !== 'back')
    render(<Delegated />)
    press('Escape')
    expect(leaveScene).toHaveBeenCalledOnce()
  })
})
