import { describe, expect, it, vi } from 'vitest'
import type { QueryClient } from '@tanstack/react-query'
import { invalidateCharacterDependents } from './character-cache'

/**
 * Pin the invalidation surface. If someone renames a query key or adds
 * a new derived view without also touching this helper, we want this
 * spec to fail loudly.
 */

function makeQc() {
  const invalidateQueries = vi.fn()
  return {
    qc: { invalidateQueries } as unknown as QueryClient,
    invalidateQueries,
  }
}

describe('invalidateCharacterDependents', () => {
  it('invalidates sheet + campaigns tab for the given character', () => {
    const { qc, invalidateQueries } = makeQc()
    invalidateCharacterDependents(qc, 42)
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['characters', 42, 'sheet'],
    })
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['characters', 42, 'campaigns'],
    })
  })

  it('invalidates every campaign members roster via predicate', () => {
    const { qc, invalidateQueries } = makeQc()
    invalidateCharacterDependents(qc, 42)
    const predicateCall = invalidateQueries.mock.calls.find(
      (c) => c[0] && typeof c[0].predicate === 'function',
    )
    expect(predicateCall).toBeDefined()
    const predicate = predicateCall![0].predicate as (q: {
      queryKey: unknown[]
    }) => boolean
    expect(predicate({ queryKey: ['campaigns', 1, 'members'] })).toBe(true)
    expect(predicate({ queryKey: ['campaigns', 99, 'members'] })).toBe(true)
    expect(predicate({ queryKey: ['campaigns', 1, 'sessions'] })).toBe(false)
    expect(predicate({ queryKey: ['characters', 1] })).toBe(false)
  })

  it('does NOT invalidate the base ["characters", id] key', () => {
    const { qc, invalidateQueries } = makeQc()
    invalidateCharacterDependents(qc, 42)
    const baseCall = invalidateQueries.mock.calls.find(
      (c) =>
        c[0] &&
        Array.isArray(c[0].queryKey) &&
        c[0].queryKey.length === 2 &&
        c[0].queryKey[0] === 'characters' &&
        c[0].queryKey[1] === 42,
    )
    expect(baseCall).toBeUndefined()
  })
})
