import { createStore, produce } from 'solid-js/store'

/**
 * How many times each limited power ("1/cena", "1/dia") has been used, per
 * character. Local-only like [[conditionals-store]]: the book limit is
 * table-trust, not server state. Cleared by the Efeitos block's "Encerrar
 * cena/dia".
 *
 * Persisted under the SAME key/shape the React zustand used
 * (`t20-power-uses` → `{ state: { uses } }`): mid-migration the player
 * alternates between the two apps and a power spent at the table must stay
 * spent.
 */
export const POWER_USES_STORAGE_KEY = 't20-power-uses'

/** Scope of a power's use limit — mirrors t20-data `ActivationUses`. */
export type PowerUseScope = 'scene' | 'day'

export type PowerUseCounts = { scene: Record<string, number>; day: Record<string, number> }

type UsesByCharacter = Record<string, PowerUseCounts>

export type PowerUsesStore = {
  /** Used counts for one power. Tracks — read it inside a memo. */
  used: (characterId: number, powerId: string) => { scene: number; day: number }
  bump: (characterId: number, powerId: string, scope: PowerUseScope) => void
  resetScene: (characterId: number) => void
  resetDay: (characterId: number) => void
}

const EMPTY_COUNTS: PowerUseCounts = { scene: {}, day: {} }

function isCountBucket(value: unknown): value is Record<string, number> {
  return (
    !!value &&
    typeof value === 'object' &&
    Object.values(value).every((n) => typeof n === 'number')
  )
}

/** Defensive read: a corrupt or older blob must not take the sheet down. */
export function readStoredPowerUses(raw: string | null): UsesByCharacter {
  if (!raw) return {}
  try {
    const parsed = (JSON.parse(raw) as { state?: { uses?: unknown } }).state?.uses
    if (!parsed || typeof parsed !== 'object') return {}
    const entries = Object.entries(parsed as Record<string, unknown>).flatMap(
      ([id, counts]): [string, PowerUseCounts][] => {
        const { scene, day } = (counts ?? {}) as Partial<PowerUseCounts>
        if (!isCountBucket(scene) || !isCountBucket(day)) return []
        return [[id, { scene, day }]]
      },
    )
    return Object.fromEntries(entries)
  } catch {
    return {}
  }
}

/**
 * @example
 * const powerUses = createPowerUsesStore()
 * powerUses.bump(character.id, 'class.barbaro.furia', 'day')
 * powerUses.used(character.id, 'class.barbaro.furia').day // 1
 */
export function createPowerUsesStore(
  storage: Storage | undefined = globalThis.localStorage,
): PowerUsesStore {
  const [uses, setUses] = createStore<UsesByCharacter>(
    readStoredPowerUses(storage?.getItem(POWER_USES_STORAGE_KEY) ?? null),
  )

  const edit = (mutate: (draft: UsesByCharacter) => void) => {
    setUses(produce(mutate))
    storage?.setItem(POWER_USES_STORAGE_KEY, JSON.stringify({ state: { uses } }))
  }

  return {
    used: (characterId, powerId) => {
      const counts = uses[String(characterId)] ?? EMPTY_COUNTS
      return { scene: counts.scene[powerId] ?? 0, day: counts.day[powerId] ?? 0 }
    },

    bump: (characterId, powerId, scope) =>
      edit((draft) => {
        const key = String(characterId)
        const current = draft[key] ?? { scene: {}, day: {} }
        draft[key] = {
          ...current,
          [scope]: { ...current[scope], [powerId]: (current[scope][powerId] ?? 0) + 1 },
        }
      }),

    resetScene: (characterId) =>
      edit((draft) => {
        const current = draft[String(characterId)]
        if (current) draft[String(characterId)] = { ...current, scene: {} }
      }),

    // Ending the day ends the running scene too (book rest semantics), so the
    // whole character entry goes.
    resetDay: (characterId) =>
      edit((draft) => {
        delete draft[String(characterId)]
      }),
  }
}
