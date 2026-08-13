import { createStore, produce } from 'solid-js/store'

/**
 * Which opt-in conditionals (Fúria, Ataque Poderoso, a homebrew toggles…) the
 * player has switched on, per character. It is a CLIENT choice, not server
 * state: the sheet recomputes against this set, so it lives here and persists
 * locally.
 *
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20-conditionals` → `{ state: { active } }`): mid-migration the player
 * alternates between the two apps and what is toggled at the table must not
 * disappear on the way.
 */
export const CONDITIONALS_STORAGE_KEY = 't20-conditionals'

type ActiveByCharacter = Record<string, string[]>

export type ConditionalsStore = {
  /** The ids switched on for one character. Tracks — read it inside a memo. */
  active: (characterId: number) => ReadonlySet<string>
  toggle: (characterId: number, id: string) => void
  /** Batch: set every id in `ids` to `value` for one character. */
  setMany: (characterId: number, ids: string[], value: boolean) => void
  clear: (characterId: number) => void
}

/** Defensive read: a corrupt or older blob must not take the sheet down. */
export function readStoredConditionals(raw: string | null): ActiveByCharacter {
  if (!raw) return {}
  try {
    const parsed = (JSON.parse(raw) as { state?: { active?: unknown } }).state?.active
    if (!parsed || typeof parsed !== 'object') return {}
    const entries = Object.entries(parsed as Record<string, unknown>).filter(
      (entry): entry is [string, string[]] =>
        Array.isArray(entry[1]) && entry[1].every((id) => typeof id === 'string'),
    )
    return Object.fromEntries(entries)
  } catch {
    return {}
  }
}

/**
 * @example
 * const conditionals = createConditionalsStore()
 * conditionals.toggle(character.id, 'furia')
 * computedSheetFor(character, conditionals.active(character.id))
 */
export function createConditionalsStore(
  storage: Storage | undefined = globalThis.localStorage,
): ConditionalsStore {
  const [active, setActive] = createStore<ActiveByCharacter>(
    readStoredConditionals(storage?.getItem(CONDITIONALS_STORAGE_KEY) ?? null),
  )

  const persist = () => {
    storage?.setItem(CONDITIONALS_STORAGE_KEY, JSON.stringify({ state: { active } }))
  }

  const edit = (mutate: (draft: ActiveByCharacter) => void) => {
    setActive(produce(mutate))
    persist()
  }

  return {
    // A fresh Set per read: callers pass it to `computedSheetFor`, which keys
    // its cache on the set's CONTENTS, and a mutable shared Set would let a
    // toggle slip past that cache unnoticed.
    active: (characterId) => new Set(active[String(characterId)] ?? []),

    toggle: (characterId, id) =>
      edit((draft) => {
        const key = String(characterId)
        const current = draft[key] ?? []
        draft[key] = current.includes(id)
          ? current.filter((x) => x !== id)
          : [...current, id]
      }),

    setMany: (characterId, ids, value) =>
      edit((draft) => {
        const key = String(characterId)
        const current = new Set(draft[key] ?? [])
        for (const id of ids) {
          if (value) current.add(id)
          else current.delete(id)
        }
        draft[key] = [...current]
      }),

    clear: (characterId) =>
      edit((draft) => {
        delete draft[String(characterId)]
      }),
  }
}
