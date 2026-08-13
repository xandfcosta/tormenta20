import { createRoot } from 'solid-js'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMediaQuery, createPrefersReducedMotion } from './media-query'

/** Named fake for a MediaQueryList, so a test can flip the match at will. */
class FakeMediaQueryList {
  readonly listeners = new Set<() => void>()
  constructor(
    readonly media: string,
    public matches: boolean,
  ) {}
  addEventListener(_type: string, fn: () => void) {
    this.listeners.add(fn)
  }
  removeEventListener(_type: string, fn: () => void) {
    this.listeners.delete(fn)
  }
  /** Simulates the viewport or OS setting changing. */
  emit(matches: boolean) {
    this.matches = matches
    for (const fn of this.listeners) fn()
  }
}

/** Installs the fake and hands back the list the code under test will see. */
function installMatchMedia(matches: boolean): { last: () => FakeMediaQueryList } {
  let created: FakeMediaQueryList | undefined
  window.matchMedia = vi.fn((media: string) => {
    created = new FakeMediaQueryList(media, matches)
    return created as unknown as MediaQueryList
  })
  return {
    last: () => {
      if (!created) throw new Error('matchMedia não foi chamado (esperado 1 chamada)')
      return created
    },
  }
}

afterEach(() => vi.restoreAllMocks())

describe('createMediaQuery', () => {
  it('começa com o valor atual da query', () => {
    installMatchMedia(true)
    createRoot((dispose) => {
      expect(createMediaQuery('(min-width: 1280px)')()).toBe(true)
      dispose()
    })
  })

  it('reage quando a query muda', () => {
    const mq = installMatchMedia(false)
    createRoot((dispose) => {
      const matches = createMediaQuery('(min-width: 1280px)')
      expect(matches()).toBe(false)
      mq.last().emit(true)
      expect(matches()).toBe(true)
      dispose()
    })
  })

  it('desliga o listener no dispose (sem vazamento entre cenas)', () => {
    const mq = installMatchMedia(false)
    createRoot((dispose) => {
      createMediaQuery('(min-width: 1280px)')
      expect(mq.last().listeners.size).toBe(1)
      dispose()
    })
    expect(mq.last().listeners.size).toBe(0)
  })

  it('createPrefersReducedMotion pergunta pela query da WCAG 2.3.3', () => {
    const mq = installMatchMedia(true)
    createRoot((dispose) => {
      expect(createPrefersReducedMotion()()).toBe(true)
      expect(mq.last().media).toBe('(prefers-reduced-motion: reduce)')
      dispose()
    })
  })
})
