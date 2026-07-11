import { describe, expect, it } from 'vitest'
import {
  characterOptionsQueryOptions,
  characterQueryOptions,
  charactersQueryOptions,
} from './queries'

/**
 * TanStack queryOptions wrappers — pin the queryKey shape (which the
 * mutation hooks invalidate against) and the immutable-catalog cache
 * control on characterOptions.
 */
describe('queryKey shapes', () => {
  it('charactersQuery uses ["characters"]', () => {
    expect(charactersQueryOptions.queryKey).toEqual(['characters'])
  })

  it('characterOptionsQuery uses ["characters", "options"]', () => {
    expect(characterOptionsQueryOptions.queryKey).toEqual([
      'characters',
      'options',
    ])
  })

  it('characterQuery(id) uses ["characters", id]', () => {
    expect(characterQueryOptions(42).queryKey).toEqual(['characters', 42])
  })
})

describe('characterOptionsQuery — cache control', () => {
  it('has staleTime=Infinity (catalog is immutable in-session)', () => {
    expect(characterOptionsQueryOptions.staleTime).toBe(Infinity)
  })
})
