import { beforeEach, describe, expect, it } from 'vitest'
import {
  STANCE_ACTIVATIONS_STORAGE_KEY,
  useStanceActivationStore,
} from './stance-activation-store'

/**
 * Per-character stance payment records. Persists via zustand/persist —
 * src/test-setup.ts installs a deterministic MemoryStorage. Reset state +
 * storage between cases to avoid cross-test leakage.
 */
beforeEach(() => {
  localStorage.clear()
  useStanceActivationStore.setState({ records: {} })
})

describe('logActivation', () => {
  it('stores the payment record per character and flag', () => {
    useStanceActivationStore
      .getState()
      .logActivation(1, 'furia', { steps: 1, pmPaid: 3 })
    expect(useStanceActivationStore.getState().records[1]).toEqual({
      furia: { steps: 1, pmPaid: 3 },
    })
  })

  it('overwrites the record on re-activation (latest payment wins)', () => {
    const store = useStanceActivationStore.getState()
    store.logActivation(1, 'furia', { steps: 0, pmPaid: 2 })
    store.logActivation(1, 'furia', { steps: 2, pmPaid: 4 })
    expect(useStanceActivationStore.getState().records[1]?.furia).toEqual({
      steps: 2,
      pmPaid: 4,
    })
  })

  it('scopes per character — logging on char 1 leaves char 2 untouched', () => {
    const store = useStanceActivationStore.getState()
    store.logActivation(1, 'furia', { steps: 0, pmPaid: 2 })
    store.logActivation(2, 'furia', { steps: 1, pmPaid: 3 })
    expect(useStanceActivationStore.getState().records[1]?.furia?.pmPaid).toBe(2)
    expect(useStanceActivationStore.getState().records[2]?.furia?.pmPaid).toBe(3)
  })
})

describe('clearActivation', () => {
  it('removes only the given flag for the character', () => {
    const store = useStanceActivationStore.getState()
    store.logActivation(1, 'furia', { steps: 0, pmPaid: 2 })
    store.logActivation(1, 'outra', { steps: 0, pmPaid: 1 })
    store.clearActivation(1, 'furia')
    expect(useStanceActivationStore.getState().records[1]).toEqual({
      outra: { steps: 0, pmPaid: 1 },
    })
  })

  it('is a no-op for unknown character or flag', () => {
    useStanceActivationStore
      .getState()
      .logActivation(2, 'furia', { steps: 0, pmPaid: 2 })
    useStanceActivationStore.getState().clearActivation(999, 'furia')
    useStanceActivationStore.getState().clearActivation(2, 'inexistente')
    expect(useStanceActivationStore.getState().records[2]?.furia).toEqual({
      steps: 0,
      pmPaid: 2,
    })
  })
})

describe('persistence — localStorage round-trip', () => {
  it('writes to the configured storage key on log', () => {
    useStanceActivationStore
      .getState()
      .logActivation(7, 'furia', { steps: 1, pmPaid: 3 })
    const raw = localStorage.getItem(STANCE_ACTIVATIONS_STORAGE_KEY)
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw!)
    expect(parsed.state.records[7].furia).toEqual({ steps: 1, pmPaid: 3 })
  })
})
