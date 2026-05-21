/**
 * Tormenta 20 panteão (PDF Cap 3, Religião). The 14 deuses maiores plus a
 * handful of deuses menores commonly chosen by player classes. The catalog
 * is used by the classChoices picker (clérigo/paladino/druida devoto) and
 * by typed `classChoice` prerequisites that gate powers on devotion.
 *
 * `paladinoEligible` / `druidaEligible` reflect the per-class devoto whitelist:
 *   Paladino — Azgher, Khalmyr, Lena, Lin-Wu, Marah, Tanna-Toh, Thyatis, Valkaria
 *   Druida   — Allihanna, Megalokk, Oceano
 *
 * Clérigos may pick any deus maior. The boolean `major` separates deuses
 * maiores from deuses menores for the clérigo picker.
 */
export type Deus = {
  id: string
  name: string
  major: boolean
  paladinoEligible: boolean
  druidaEligible: boolean
}

export const DEUSES: Deus[] = [
  { id: 'allihanna', name: 'Allihanna', major: true, paladinoEligible: false, druidaEligible: true },
  { id: 'arsenal', name: 'Arsenal', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'azgher', name: 'Azgher', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'hyninn', name: 'Hyninn', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'kallyadranoch', name: 'Kallyadranoch', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'khalmyr', name: 'Khalmyr', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'lena', name: 'Lena', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'lin-wu', name: 'Lin-Wu', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'marah', name: 'Marah', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'megalokk', name: 'Megalokk', major: true, paladinoEligible: false, druidaEligible: true },
  { id: 'nimb', name: 'Nimb', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'oceano', name: 'Oceano', major: true, paladinoEligible: false, druidaEligible: true },
  { id: 'sszzaas', name: 'Sszzaas', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'tanna-toh', name: 'Tanna-Toh', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'tenebra', name: 'Tenebra', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'thwor', name: 'Thwor', major: true, paladinoEligible: false, druidaEligible: false },
  { id: 'thyatis', name: 'Thyatis', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'valkaria', name: 'Valkaria', major: true, paladinoEligible: true, druidaEligible: false },
  { id: 'wynna', name: 'Wynna', major: true, paladinoEligible: false, druidaEligible: false },
]

export const DEUS_BY_ID: Record<string, Deus> = Object.fromEntries(
  DEUSES.map((d) => [d.id, d]),
)

/** Caminhos per class — subpath choices unlocked by class abilities. */
export type CaminhoOption = { id: string; name: string }

export const CAMINHOS: Record<string, CaminhoOption[]> = {
  Arcanista: [
    { id: 'bruxo', name: 'Bruxo' },
    { id: 'feiticeiro', name: 'Feiticeiro' },
    { id: 'mago', name: 'Mago' },
  ],
  Paladino: [
    { id: 'egide-sagrada', name: 'Égide Sagrada' },
    { id: 'montaria-sagrada', name: 'Montaria Sagrada' },
  ],
  Cavaleiro: [
    { id: 'bastiao', name: 'Bastião' },
    { id: 'montaria', name: 'Montaria' },
  ],
}

/**
 * Per-class persisted choices keyed by className. Stored as JSON on
 * Character.classChoices and merged into prerequisite evaluation.
 */
export type ClassChoiceBlob = {
  /** Deus id chosen as devoto (clérigo/paladino/druida). */
  devoto?: string
  /** Caminho/subpath id (arcanista, paladino L5, cavaleiro L5, ...). */
  caminho?: string
}

export type ClassChoices = Partial<Record<string, ClassChoiceBlob>>

/**
 * Returns the deus catalog filtered for the given class's devoto picker,
 * or null when the class has no devoto slot. Mirrors the per-class lists
 * in PDF Cap 3 (Religião) — Clérigo picks any deus maior; Paladino and
 * Druida have narrower whitelists encoded as boolean flags per deus.
 */
export function devotoOptionsFor(className: string): Deus[] | null {
  switch (className) {
    case 'Clérigo':
      return DEUSES.filter((d) => d.major)
    case 'Paladino':
      return DEUSES.filter((d) => d.paladinoEligible)
    case 'Druida':
      return DEUSES.filter((d) => d.druidaEligible)
    default:
      return null
  }
}

/**
 * Per-class caminho/subpath slot. Returns options + the class level at
 * which the caminho choice unlocks (the class's "Caminho" auto-power), or
 * null when the class has no caminho slot.
 */
export function caminhoSlotFor(
  className: string,
): { options: CaminhoOption[]; minLevel: number } | null {
  const options = CAMINHOS[className]
  if (!options) return null
  const minLevel = className === 'Arcanista' ? 1 : 5
  return { options, minLevel }
}
