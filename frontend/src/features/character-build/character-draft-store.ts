import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { RaceChoiceState } from './grant-helpers'
import { type CharacterFormValues, wizardDefaults } from './wizard-steps'

export const CHARACTER_DRAFT_STORAGE_KEY = 't20:new-character-draft'

/**
 * Persisted draft for the multi-page creation wizard. Holds the in-progress
 * form values + the out-of-band race attribute choices (floating +1s /
 * subrace) so a refresh or leaving mid-flow survives. The TanStack Form in the
 * wizard shell hydrates from `values` and mirrors edits back via `setValues`;
 * `reset` clears the draft on successful create or cancel.
 */
type CharacterDraftState = {
  values: CharacterFormValues
  raceChoices: RaceChoiceState
  setValues: (values: CharacterFormValues) => void
  setRaceChoices: (raceChoices: RaceChoiceState) => void
  reset: () => void
}

export const useCharacterDraftStore = create<CharacterDraftState>()(
  persist(
    (set) => ({
      values: wizardDefaults,
      raceChoices: {},
      setValues: (values) => set({ values }),
      setRaceChoices: (raceChoices) => set({ raceChoices }),
      reset: () => set({ values: wizardDefaults, raceChoices: {} }),
    }),
    { name: CHARACTER_DRAFT_STORAGE_KEY },
  ),
)

/** Draft has at least one meaningful choice (for a "continuar rascunho" prompt). */
export function hasResumableDraft(): boolean {
  const { values } = useCharacterDraftStore.getState()
  return values.races.length > 0 || values.classes.length > 0
}
