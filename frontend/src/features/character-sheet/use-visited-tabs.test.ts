import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { useVisitedTabs } from './use-visited-tabs'

describe('useVisitedTabs', () => {
  it('starts with the initial active tab already visited', () => {
    const { result } = renderHook(() => useVisitedTabs('expertises'))
    expect([...result.current]).toEqual(['expertises'])
  })

  it('accumulates every visited tab and never drops one', () => {
    const { result, rerender } = renderHook(({ t }) => useVisitedTabs(t), {
      initialProps: { t: 'expertises' },
    })
    rerender({ t: 'abilities' })
    rerender({ t: 'spells' })
    rerender({ t: 'expertises' }) // revisit — no duplicate, nothing dropped
    expect(result.current.has('expertises')).toBe(true)
    expect(result.current.has('abilities')).toBe(true)
    expect(result.current.has('spells')).toBe(true)
    expect(result.current.size).toBe(3)
  })

  it('keeps a stable set reference when the active tab is already visited', () => {
    const { result, rerender } = renderHook(({ t }) => useVisitedTabs(t), {
      initialProps: { t: 'expertises' },
    })
    const first = result.current
    act(() => rerender({ t: 'expertises' }))
    expect(result.current).toBe(first) // no needless re-render churn
  })
})
