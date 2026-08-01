import { DEUSES, devotoOptionsFor } from '@tormenta20/t20-data'

const DEUS_ID_BY_NAME = new Map(DEUSES.map((d) => [d.name, d.id]))

/** Shape the wizard form stores under `classChoices` (zod schema, not the
 *  t20-data Partial-record alias — TanStack's Updater needs the exact type). */
type WizardClassChoices = Record<string, { devoto?: string; caminho?: string }>

type DevocaoValues = {
  god?: string
  classes?: { className: string }[]
  classChoices?: WizardClassChoices
}

/**
 * Unified devoção (book p96 — one devotion per character): the Identidade god
 * drives the per-class devoto slot. Returns the corrected classChoices when a
 * class's devoto diverges from the chosen god (and the god is valid for that
 * class's list), or null when nothing needs to change. No god set → no sync,
 * so the class-side sentinels (Panteão, Paladino do Bem) stay usable.
 */
export function devotoSyncPatch(
  values: DevocaoValues,
): WizardClassChoices | null {
  const deusId = values.god ? DEUS_ID_BY_NAME.get(values.god) : undefined
  if (!deusId) return null
  let next: WizardClassChoices | null = null
  for (const c of values.classes ?? []) {
    const options = devotoOptionsFor(c.className)
    if (!options?.some((d) => d.id === deusId)) continue
    const current = values.classChoices?.[c.className]?.devoto
    if (current === deusId) continue
    next = next ?? { ...(values.classChoices ?? {}) }
    next[c.className] = { ...next[c.className], devoto: deusId }
  }
  return next
}

/** Apply the sync onto the wizard form (no-op when already in sync). */
export function syncDevotoFromGod(form: {
  state: { values: unknown }
  setFieldValue: (name: 'classChoices', value: WizardClassChoices) => void
}): void {
  const patch = devotoSyncPatch(form.state.values as DevocaoValues)
  if (patch) form.setFieldValue('classChoices', patch)
}
