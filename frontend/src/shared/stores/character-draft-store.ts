import { type Accessor, createSignal } from 'solid-js'
import { createStore, produce, reconcile } from 'solid-js/store'
import type { RaceChoice, RaceChoiceState } from '@/features/character-build/grant-helpers'
import { type CharacterFormValues, wizardDefaults } from '@/features/character-build/wizard-steps'

/**
 * Persisted under the SAME key/shape the React app's zustand store used
 * (`t20:new-character-draft` → `{ state: { values, raceChoices, attributeMode } }`):
 * mid-migration a player may start the Forja in one app and finish in the
 * other, and half a character is not something to lose to a port.
 */
export const CHARACTER_DRAFT_STORAGE_KEY = 't20:new-character-draft'

/** Attribute-editing mode: free edit (default) or book point-buy (p17). */
export type AttributeMode = 'free' | 'point-buy'

type DraftSnapshot = {
  values: CharacterFormValues
  raceChoices: RaceChoiceState
  attributeMode: AttributeMode
}

export type CharacterDraftStore = {
  /** Store proxy — read fields directly (`draft.values.name`) so a step that
   *  touches one field doesn't re-run every other step's memos. */
  values: CharacterFormValues
  raceChoices: RaceChoiceState
  attributeMode: Accessor<AttributeMode>
  setValue: <K extends keyof CharacterFormValues>(
    key: K,
    value: CharacterFormValues[K],
  ) => void
  patchValues: (patch: Partial<CharacterFormValues>) => void
  setRaceChoice: (raceName: string, choice: RaceChoice) => void
  setAttributeMode: (mode: AttributeMode) => void
  reset: () => void
}

const isPlainObject = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null && !Array.isArray(v)

/**
 * Keep a stored field only when its shape still matches the default's. A blob
 * written by an older version (or hand-edited) must not put a string where the
 * wizard expects an array: the failure would surface deep inside a step, far
 * from the cause.
 */
function sameShape(stored: unknown, fallback: unknown): boolean {
  if (Array.isArray(fallback)) return Array.isArray(stored)
  if (isPlainObject(fallback)) return isPlainObject(stored)
  return typeof stored === typeof fallback
}

function mergeValues(stored: unknown): CharacterFormValues {
  if (!isPlainObject(stored)) return { ...wizardDefaults }
  // Accumulate untyped: indexing CharacterFormValues by the whole key union
  // narrows the assignment target to `never`. Shape is checked per key above,
  // so the single cast on the way out is what `sameShape` earns.
  const out: Record<string, unknown> = { ...wizardDefaults }
  for (const [key, fallback] of Object.entries(wizardDefaults)) {
    const value = stored[key]
    if (value !== undefined && sameShape(value, fallback)) out[key] = value
  }
  return out as CharacterFormValues
}

/** Defensive read: a corrupt or older blob must not take the Forja down. */
export function readStoredDraft(raw: string | null): DraftSnapshot {
  const empty: DraftSnapshot = {
    values: { ...wizardDefaults },
    raceChoices: {},
    attributeMode: 'free',
  }
  if (!raw) return empty
  try {
    const state = (JSON.parse(raw) as { state?: unknown }).state
    if (!isPlainObject(state)) return empty
    return {
      values: mergeValues(state.values),
      raceChoices: isPlainObject(state.raceChoices)
        ? (state.raceChoices as RaceChoiceState)
        : {},
      attributeMode: state.attributeMode === 'point-buy' ? 'point-buy' : 'free',
    }
  } catch {
    return empty
  }
}

/**
 * The in-progress character of the creation wizard, surviving a refresh or a
 * walk away mid-flow. Holds the form values plus the out-of-band race
 * attribute choices (floating +1s / ascendência), which are not form fields.
 *
 * @example
 * const draft = createCharacterDraftStore()
 * draft.setValue('races', ['Anão'])
 * draft.setRaceChoice('Humano', { floatingPicks: ['strength'] })
 */
export function createCharacterDraftStore(
  storage: Storage | undefined = globalThis.localStorage,
): CharacterDraftStore {
  const initial = readStoredDraft(storage?.getItem(CHARACTER_DRAFT_STORAGE_KEY) ?? null)
  const [values, setValues] = createStore<CharacterFormValues>(initial.values)
  const [raceChoices, setRaceChoices] = createStore<RaceChoiceState>(initial.raceChoices)
  const [attributeMode, setAttributeModeSignal] = createSignal(initial.attributeMode)

  const persist = () => {
    const state: DraftSnapshot = { values, raceChoices, attributeMode: attributeMode() }
    storage?.setItem(CHARACTER_DRAFT_STORAGE_KEY, JSON.stringify({ state }))
  }

  return {
    values,
    raceChoices,
    attributeMode,

    setValue: (key, value) => {
      setValues(key, value as never)
      persist()
    },

    patchValues: (patch) => {
      setValues(patch as Partial<CharacterFormValues>)
      persist()
    },

    setRaceChoice: (raceName, choice) => {
      setRaceChoices(raceName, choice)
      persist()
    },

    setAttributeMode: (mode) => {
      setAttributeModeSignal(mode)
      persist()
    },

    reset: () => {
      // `reconcile` instead of a fresh store: the wizard holds THIS proxy, so
      // replacing the object would leave every mounted step reading a corpse.
      setValues(reconcile({ ...wizardDefaults }))
      setRaceChoices(produce((draft) => {
        for (const key of Object.keys(draft)) delete draft[key]
      }))
      setAttributeModeSignal('free')
      storage?.removeItem(CHARACTER_DRAFT_STORAGE_KEY)
    },
  }
}
