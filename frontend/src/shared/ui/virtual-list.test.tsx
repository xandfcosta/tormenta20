import { render, screen } from '@testing-library/react'
import { beforeAll, describe, expect, it } from 'vitest'
import { VirtualList } from './virtual-list'

/**
 * jsdom has no layout engine or ResizeObserver, so react-virtual can't measure
 * a real viewport and would render zero rows. A no-op ResizeObserver plus a
 * fixed 256px-tall getBoundingClientRect give the virtualizer a viewport so it
 * renders the leading window — enough to prove the wrapper virtualizes (top
 * rows present, tail rows not).
 */
class NoopResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

beforeAll(() => {
  globalThis.ResizeObserver =
    NoopResizeObserver as unknown as typeof ResizeObserver
  Element.prototype.getBoundingClientRect = () =>
    ({ width: 320, height: 256, top: 0, left: 0, right: 320, bottom: 256 }) as DOMRect
  // react-virtual sizes the element scroller from clientHeight, which jsdom
  // reports as 0 without a layout engine.
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    value: 256,
  })
})

const ITEMS = Array.from({ length: 200 }, (_, i) => ({
  id: i,
  name: `Item ${i}`,
}))

describe('VirtualList', () => {
  it('virtualizes: reserves full height but does not render the whole list', () => {
    const { container } = render(
      <VirtualList
        className="h-64"
        items={ITEMS}
        estimateSize={40}
        getKey={(it) => it.id}
        renderItem={(it) => <span>{it.name}</span>}
      />,
    )
    // Scroll container carries the caller's className + owns the scroll.
    const scroll = container.firstElementChild as HTMLElement
    expect(scroll.className).toContain('overflow-y-auto')
    expect(scroll.className).toContain('h-64')
    // Spacer reserves scroll height for every item (200 × 40 = 8000px).
    const spacer = scroll.firstElementChild as HTMLElement
    expect(spacer.style.height).toBe('8000px')
    // A tail item far past the viewport is never in the DOM.
    expect(screen.queryByText('Item 199')).not.toBeInTheDocument()
    expect(
      container.querySelectorAll('[data-index]').length,
    ).toBeLessThan(ITEMS.length)
  })

  it('renders no rows when empty', () => {
    const { container } = render(
      <VirtualList
        items={[]}
        getKey={(it: { id: number }) => it.id}
        renderItem={() => <span>never</span>}
      />,
    )
    expect(screen.queryByText('never')).not.toBeInTheDocument()
    expect(container.querySelector('[data-index]')).toBeNull()
  })
})
