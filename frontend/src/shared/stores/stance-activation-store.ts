import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type CharacterId = number

/**
 * What the player paid entering a stance. `steps` are the extra-PM stepper
 * picks (Fúria p40: +1 PM per +1 bônus every 5 levels). The stepped EXTRA has
 * no engine modifier yet (spell-engine-deferred style), so this record is the
 * display-only source of truth for "+N extra (stepper)" in Posturas ativas.
 */
export type StanceActivationRecord = { steps: number; pmPaid: number }

type StanceActivationState = {
  records: Record<CharacterId, Record<string, StanceActivationRecord>>
  logActivation: (
    characterId: CharacterId,
    flag: string,
    record: StanceActivationRecord,
  ) => void
  clearActivation: (characterId: CharacterId, flag: string) => void
}

export const STANCE_ACTIVATIONS_STORAGE_KEY = 't20-stance-activations'

/**
 * Per-character record of active-stance payments — local-only like
 * `useConditionalsStore` (the flag state itself lives there; this only
 * remembers what was paid, so ending a stance never refunds).
 *
 * @example useStanceActivationStore.getState().logActivation(1, 'furia', { steps: 1, pmPaid: 3 })
 */
export const useStanceActivationStore = create<StanceActivationState>()(
  persist(
    (set) => ({
      records: {},
      logActivation: (characterId, flag, record) =>
        set((s) => ({
          records: {
            ...s.records,
            [characterId]: { ...s.records[characterId], [flag]: record },
          },
        })),
      clearActivation: (characterId, flag) =>
        set((s) => {
          const current = s.records[characterId]
          if (!current?.[flag]) return s
          const next = { ...current }
          delete next[flag]
          return { records: { ...s.records, [characterId]: next } }
        }),
    }),
    { name: STANCE_ACTIVATIONS_STORAGE_KEY },
  ),
)

/**
 * Live payment record for one active stance, or undefined when the stance was
 * never activated through the new path (legacy toggle / cleared storage).
 *
 * @example const paid = useStanceActivation(1, 'furia') // { steps: 1, pmPaid: 3 }
 */
export function useStanceActivation(
  characterId: CharacterId,
  flag: string,
): StanceActivationRecord | undefined {
  return useStanceActivationStore((s) => s.records[characterId]?.[flag])
}
