import { beforeEach, describe, expect, it } from 'vitest'
import {
  POWER_USES_STORAGE_KEY,
  usePowerUsesStore,
} from './power-uses-store'

/**
 * Per-character limited-power-use counters. Persists to localStorage via
 * zustand/persist; src/test-setup.ts installs a deterministic MemoryStorage
 * so persist captures a working handle at module init. Reset state + storage
 * between cases to avoid cross-test leakage.
 */
beforeEach(() => {
  localStorage.clear()
  usePowerUsesStore.setState({ uses: {} })
})

describe('bump', () => {
  it('starts a power at 1 on first use', () => {
    usePowerUsesStore.getState().bump(1, 'class.barbaro.golpe-poderoso', 'scene')
    expect(
      usePowerUsesStore.getState().uses[1]!.scene['class.barbaro.golpe-poderoso'],
    ).toBe(1)
  })

  it('increments on repeated use', () => {
    usePowerUsesStore.getState().bump(1, 'p', 'day')
    usePowerUsesStore.getState().bump(1, 'p', 'day')
    expect(usePowerUsesStore.getState().uses[1]!.day.p).toBe(2)
  })

  it('keeps scene and day buckets independent', () => {
    usePowerUsesStore.getState().bump(1, 'p', 'scene')
    usePowerUsesStore.getState().bump(1, 'p', 'day')
    expect(usePowerUsesStore.getState().uses[1]).toEqual({
      scene: { p: 1 },
      day: { p: 1 },
    })
  })

  it('scopes per character — bumping char 1 leaves char 2 untouched', () => {
    usePowerUsesStore.getState().bump(1, 'a', 'scene')
    usePowerUsesStore.getState().bump(2, 'b', 'scene')
    expect(usePowerUsesStore.getState().uses[1]!.scene).toEqual({ a: 1 })
    expect(usePowerUsesStore.getState().uses[2]!.scene).toEqual({ b: 1 })
  })
})

describe('resetScene', () => {
  it('clears scene counters but keeps day counters', () => {
    usePowerUsesStore.getState().bump(1, 'p', 'scene')
    usePowerUsesStore.getState().bump(1, 'q', 'day')
    usePowerUsesStore.getState().resetScene(1)
    expect(usePowerUsesStore.getState().uses[1]).toEqual({
      scene: {},
      day: { q: 1 },
    })
  })

  it('is a no-op for unknown character ids', () => {
    usePowerUsesStore.getState().bump(2, 'p', 'scene')
    usePowerUsesStore.getState().resetScene(999)
    expect(usePowerUsesStore.getState().uses[2]!.scene).toEqual({ p: 1 })
  })
})

describe('resetDay', () => {
  it('clears both day AND scene counters (ending the day ends the scene)', () => {
    usePowerUsesStore.getState().bump(1, 'p', 'scene')
    usePowerUsesStore.getState().bump(1, 'q', 'day')
    usePowerUsesStore.getState().resetDay(1)
    expect(usePowerUsesStore.getState().uses[1]).toBeUndefined()
  })

  it('leaves other characters untouched', () => {
    usePowerUsesStore.getState().bump(1, 'p', 'day')
    usePowerUsesStore.getState().bump(2, 'q', 'day')
    usePowerUsesStore.getState().resetDay(1)
    expect(usePowerUsesStore.getState().uses[2]!.day).toEqual({ q: 1 })
  })
})

describe('persistence — localStorage round-trip', () => {
  it('writes to the configured storage key on bump', () => {
    usePowerUsesStore.getState().bump(7, 'p', 'scene')
    const raw = localStorage.getItem(POWER_USES_STORAGE_KEY)
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw!)
    expect(parsed.state.uses[7].scene.p).toBe(1)
  })
})
