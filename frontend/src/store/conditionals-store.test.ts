import { beforeEach, describe, expect, it } from 'vitest'
import {
  CONDITIONALS_STORAGE_KEY,
  useConditionalsStore,
} from './conditionals-store'

/**
 * Per-character conditional opt-in store. Persists to localStorage via
 * zustand/persist; src/test-setup.ts installs a deterministic
 * MemoryStorage so the store's persist captures a working handle at
 * module init. Reset state + storage between cases to avoid cross-test
 * leakage.
 */
beforeEach(() => {
  localStorage.clear()
  useConditionalsStore.setState({ active: {} })
})

describe('toggle', () => {
  it('adds an id when absent', () => {
    useConditionalsStore.getState().toggle(1, 'aco-rubi::dmg')
    expect(useConditionalsStore.getState().active[1]).toEqual([
      'aco-rubi::dmg',
    ])
  })

  it('removes an id when present', () => {
    useConditionalsStore.setState({ active: { 1: ['aco-rubi::dmg', 'x'] } })
    useConditionalsStore.getState().toggle(1, 'aco-rubi::dmg')
    expect(useConditionalsStore.getState().active[1]).toEqual(['x'])
  })

  it('scopes per character — toggling on char 1 leaves char 2 untouched', () => {
    useConditionalsStore.getState().toggle(1, 'a')
    useConditionalsStore.getState().toggle(2, 'b')
    expect(useConditionalsStore.getState().active).toEqual({
      1: ['a'],
      2: ['b'],
    })
  })
})

describe('setMany', () => {
  it('adds every id in batch when value=true', () => {
    useConditionalsStore.getState().setMany(1, ['a', 'b', 'c'], true)
    expect(useConditionalsStore.getState().active[1]!.sort()).toEqual([
      'a',
      'b',
      'c',
    ])
  })

  it('removes every id in batch when value=false', () => {
    useConditionalsStore.setState({ active: { 1: ['a', 'b', 'c'] } })
    useConditionalsStore.getState().setMany(1, ['a', 'c'], false)
    expect(useConditionalsStore.getState().active[1]).toEqual(['b'])
  })

  it('is idempotent — adding an existing id does not duplicate', () => {
    useConditionalsStore.setState({ active: { 1: ['a'] } })
    useConditionalsStore.getState().setMany(1, ['a', 'b'], true)
    expect(useConditionalsStore.getState().active[1]!.sort()).toEqual([
      'a',
      'b',
    ])
  })
})

describe('clear', () => {
  it('removes the character entry entirely', () => {
    useConditionalsStore.setState({
      active: { 1: ['a'], 2: ['b'] },
    })
    useConditionalsStore.getState().clear(1)
    expect(useConditionalsStore.getState().active).toEqual({ 2: ['b'] })
  })

  it('is a no-op for unknown character ids', () => {
    useConditionalsStore.setState({ active: { 2: ['b'] } })
    useConditionalsStore.getState().clear(999)
    expect(useConditionalsStore.getState().active).toEqual({ 2: ['b'] })
  })
})

describe('persistence — localStorage round-trip', () => {
  it('writes to the configured storage key on toggle', () => {
    useConditionalsStore.getState().toggle(7, 'flag')
    const raw = localStorage.getItem(CONDITIONALS_STORAGE_KEY)
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw!)
    expect(parsed.state.active[7]).toEqual(['flag'])
  })
})
