import { createStore, produce } from 'solid-js/store'

/**
 * What the player paid entering a stance. `steps` are the extra-PM stepper
 * picks (Fúria p40: +1 PM per +1 bônus every 5 levels). The stepped EXTRA has
 * no engine modifier yet, so this record is the display-only source of truth
 * for "+N extra (stepper)" in Posturas ativas.
 */
export type StanceActivationRecord = { steps: number; pmPaid: number }

type RecordsByCharacter = Record<string, Record<string, StanceActivationRecord>>

/**
 * Per-character record of active-stance payments — local-only like
 * [[conditionals-store]]: the flag state itself lives there, this only
 * remembers what was paid, so ending a stance never refunds.
 *
 * Persisted under the SAME key/shape the React zustand used
 * (`t20-stance-activations` → `{ state: { records } }`): mid-migration the
 * player alternates between the two apps and a Fúria paid at the table must
 * still read as paid.
 */
export const STANCE_ACTIVATIONS_STORAGE_KEY = 't20-stance-activations'

export type StanceActivationStore = {
  /** What was paid for one active stance, or undefined if it never was. */
  paidFor: (characterId: number, flag: string) => StanceActivationRecord | undefined
  logActivation: (characterId: number, flag: string, record: StanceActivationRecord) => void
  clearActivation: (characterId: number, flag: string) => void
}

function isRecord(value: unknown): value is StanceActivationRecord {
  const { steps, pmPaid } = (value ?? {}) as Partial<StanceActivationRecord>
  return typeof steps === 'number' && typeof pmPaid === 'number'
}

/** Defensive read: a corrupt or older blob must not take the sheet down. */
export function readStoredStanceActivations(raw: string | null): RecordsByCharacter {
  if (!raw) return {}
  try {
    const parsed = (JSON.parse(raw) as { state?: { records?: unknown } }).state?.records
    if (!parsed || typeof parsed !== 'object') return {}
    const entries = Object.entries(parsed as Record<string, unknown>).flatMap(
      ([id, byFlag]): [string, Record<string, StanceActivationRecord>][] => {
        if (!byFlag || typeof byFlag !== 'object') return []
        const clean = Object.entries(byFlag as Record<string, unknown>).filter(
          (entry): entry is [string, StanceActivationRecord] => isRecord(entry[1]),
        )
        return clean.length ? [[id, Object.fromEntries(clean)]] : []
      },
    )
    return Object.fromEntries(entries)
  } catch {
    return {}
  }
}

/**
 * @example
 * const stances = createStanceActivationStore()
 * stances.logActivation(character.id, 'furia', { steps: 1, pmPaid: 3 })
 */
export function createStanceActivationStore(
  storage: Storage | undefined = globalThis.localStorage,
): StanceActivationStore {
  const [records, setRecords] = createStore<RecordsByCharacter>(
    readStoredStanceActivations(storage?.getItem(STANCE_ACTIVATIONS_STORAGE_KEY) ?? null),
  )

  const edit = (mutate: (draft: RecordsByCharacter) => void) => {
    setRecords(produce(mutate))
    storage?.setItem(STANCE_ACTIVATIONS_STORAGE_KEY, JSON.stringify({ state: { records } }))
  }

  return {
    paidFor: (characterId, flag) => records[String(characterId)]?.[flag],

    logActivation: (characterId, flag, record) =>
      edit((draft) => {
        const key = String(characterId)
        draft[key] = { ...draft[key], [flag]: record }
      }),

    clearActivation: (characterId, flag) =>
      edit((draft) => {
        const current = draft[String(characterId)]
        if (current?.[flag]) delete current[flag]
      }),
  }
}
