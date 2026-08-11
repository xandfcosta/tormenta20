import { render } from '@solidjs/testing-library'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { VirtualList } from './virtual-list'

type Row = { id: string; name: string }

const rows: Row[] = Array.from({ length: 400 }, (_, i) => ({
  id: `row-${i}`,
  name: `Item ${i}`,
}))

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

/**
 * jsdom gives every element a zero-height rect, so the virtualizer's window is
 * empty here and NO row is rendered — which is exactly why the crash this list
 * caused in the catálogo dialog (index -1 → `items[-1].id`) could only be seen
 * in a real browser. What is asserted below is what jsdom can honestly answer:
 * the reserved scroll height and the out-of-range guard. The rendering itself
 * is covered by `e2e/tests/character-bag.spec.ts`.
 */
describe('VirtualList', () => {
  it('reserva a altura total da lista para o scroll', () => {
    const { container } = render(() => (
      <VirtualList
        class="max-h-56"
        items={rows}
        estimateSize={34}
        getKey={(row) => row.id}
        renderItem={(row) => <button type="button">{row.name}</button>}
      />
    ))
    const spacer = container.querySelector<HTMLElement>('.relative')
    expect(spacer?.style.height).toBe(`${400 * 34}px`)
  })

  it('lista vazia não pergunta a chave de item nenhum', () => {
    const getKey = vi.fn((row: Row) => row.id)
    render(() => (
      <VirtualList items={[]} getKey={getKey} renderItem={(row: Row) => <span>{row.name}</span>} />
    ))
    expect(getKey).not.toHaveBeenCalled()
  })
})
