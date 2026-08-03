import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type CharacterId = number

/** Scope of a power's use limit — mirrors t20-data `ActivationUses` 'cena'/'dia'. */
export type PowerUseScope = 'scene' | 'day'

type PowerUseCounts = {
  scene: Record<string, number>
  day: Record<string, number>
}

type PowerUsesState = {
  uses: Record<CharacterId, PowerUseCounts>
  bump: (
    characterId: CharacterId,
    powerId: string,
    scope: PowerUseScope,
  ) => void
  resetScene: (characterId: CharacterId) => void
  resetDay: (characterId: CharacterId) => void
}

export const POWER_USES_STORAGE_KEY = 't20-power-uses'

const EMPTY_COUNTS: PowerUseCounts = { scene: {}, day: {} }

/**
 * Per-character counters of limited power uses ("1/cena", "1/dia") — local-only
 * like `useConditionalsStore`: the book limit is table-trust, not server state.
 * Cleared by the Efeitos tab's "Encerrar cena/dia" actions.
 */
export const usePowerUsesStore = create<PowerUsesState>()(
  persist(
    (set) => ({
      uses: {},
      bump: (characterId, powerId, scope) =>
        set((s) => {
          const current = s.uses[characterId] ?? EMPTY_COUNTS
          const bucket = {
            ...current[scope],
            [powerId]: (current[scope][powerId] ?? 0) + 1,
          }
          return {
            uses: {
              ...s.uses,
              [characterId]: { ...current, [scope]: bucket },
            },
          }
        }),
      resetScene: (characterId) =>
        set((s) => {
          const current = s.uses[characterId]
          if (!current) return s
          return {
            uses: { ...s.uses, [characterId]: { ...current, scene: {} } },
          }
        }),
      // Ending the day also ends the running scene (book rest semantics), so
      // the whole character entry goes away.
      resetDay: (characterId) =>
        set((s) => {
          if (!s.uses[characterId]) return s
          const next = { ...s.uses }
          delete next[characterId]
          return { uses: next }
        }),
    }),
    { name: POWER_USES_STORAGE_KEY },
  ),
)

/**
 * Live used-counts for one power. Selects primitives (not a fresh object) per
 * subscription so re-renders only fire when this power's counters change.
 *
 * @example const { usedScene } = usePowerUsedCounts(1, 'class.barbaro.golpe-poderoso')
 */
export function usePowerUsedCounts(
  characterId: CharacterId,
  powerId: string,
): { usedScene: number; usedDay: number } {
  const usedScene = usePowerUsesStore(
    (s) => s.uses[characterId]?.scene[powerId] ?? 0,
  )
  const usedDay = usePowerUsesStore(
    (s) => s.uses[characterId]?.day[powerId] ?? 0,
  )
  return { usedScene, usedDay }
}
