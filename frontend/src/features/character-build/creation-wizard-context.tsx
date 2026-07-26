import { createContext, useContext } from 'react'
import type { CharacterOptions } from '@/shared/api/api'
import type { RaceChoiceState } from './grant-helpers'

// TanStack Form's API type is heavily generic; `any` keeps the shared form
// usable across step components without threading its type parameters
// (mirrors ClassEntryRow / NumberField / CharacterPreviewRail).
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormApi = any
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type FieldApi = any

/**
 * Shared creation-wizard state, provided once by the layout shell (which owns
 * the single TanStack Form instance) and consumed by every step rendered in
 * the router Outlet. The form survives step navigation because the shell never
 * unmounts.
 */
export type CreationWizard = {
  form: FormApi
  options: CharacterOptions
  raceChoices: RaceChoiceState
  setRaceChoices: (next: RaceChoiceState) => void
  formError: string | null
  submit: () => void
  cancel: () => void
}

const CreationWizardContext = createContext<CreationWizard | null>(null)

export const CreationWizardProvider = CreationWizardContext.Provider

export function useCreationWizard(): CreationWizard {
  const ctx = useContext(CreationWizardContext)
  if (!ctx) {
    throw new Error('useCreationWizard must be used within the wizard shell')
  }
  return ctx
}
