import { act, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useMediaQuery, usePrefersReducedMotion } from './use-media-query'

/**
 * Models the slice of MediaQueryList the hook touches — `matches` plus a
 * change listener we can fire — so the test drives a viewport flip without
 * a real browser. Named fake per the project's I/O-mock rule.
 */
class FakeMediaQueryList {
  matches: boolean
  private listeners = new Set<() => void>()
  constructor(matches: boolean) {
    this.matches = matches
  }
  addEventListener(_type: 'change', cb: () => void) {
    this.listeners.add(cb)
  }
  removeEventListener(_type: 'change', cb: () => void) {
    this.listeners.delete(cb)
  }
  emitChange(matches: boolean) {
    this.matches = matches
    for (const cb of this.listeners) cb()
  }
}

function installMatchMedia(list: FakeMediaQueryList) {
  Object.defineProperty(window, 'matchMedia', {
    value: vi.fn(() => list),
    writable: true,
    configurable: true,
  })
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('useMediaQuery', () => {
  it('returns the current match state', () => {
    installMatchMedia(new FakeMediaQueryList(true))
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'))
    expect(result.current).toBe(true)
  })

  it('re-renders when the query starts/stops matching', () => {
    const list = new FakeMediaQueryList(false)
    installMatchMedia(list)
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'))
    expect(result.current).toBe(false)
    act(() => list.emitChange(true))
    expect(result.current).toBe(true)
  })
})

describe('usePrefersReducedMotion', () => {
  it('reflects the reduced-motion media query', () => {
    const list = new FakeMediaQueryList(true)
    installMatchMedia(list)
    const { result } = renderHook(() => usePrefersReducedMotion())
    expect(result.current).toBe(true)
    act(() => list.emitChange(false))
    expect(result.current).toBe(false)
  })
})
