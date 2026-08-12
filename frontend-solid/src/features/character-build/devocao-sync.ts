import { deuses, devotoOptionsFor } from '@/shared/lib/abilities-cache'

/** Shape the wizard draft stores under `classChoices` (the zod schema's type,
 *  not the t20-data Partial-record alias, which is wider). */
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
  // Resolve god name → id off the primed catalog (warm by the loader gate).
  const deusId = values.god
    ? deuses().find((d) => d.name === values.god)?.id
    : undefined
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
