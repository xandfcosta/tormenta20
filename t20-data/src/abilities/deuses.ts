/**
 * Tormenta 20 panteão (PDF Cap 3, Religião — Tabela 1-20, p97). The 20
 * deuses maiores of Arton. The catalog is used by the classChoices picker
 * (clérigo/paladino/druida devoto) and by typed `classChoice` prerequisites
 * that gate powers on devotion.
 *
 * `paladinoEligible` / `druidaEligible` reflect the per-class devoto whitelist:
 *   Paladino (p82) — Azgher, Khalmyr, Lena, Lin-Wu, Marah, Tanna-Toh, Thyatis, Valkaria
 *   Druida   (p61) — Allihanna, Megalokk, Oceano
 *
 * Clérigos may pick any deus maior (p57). The boolean `major` separates
 * deuses maiores from deuses menores for the clérigo picker — currently all
 * entries are maiores; deuses menores arrive in future supplements (p105).
 */
export type Deus = {
  id: string
  name: string
  major: boolean
  paladinoEligible: boolean
  druidaEligible: boolean
}

/**
 * Sentinel ids used in `ClassChoiceBlob.devoto` for the non-divindade
 * alternatives both classes allow:
 *  - Clérigo (p57): "cultuar o Panteão como um todo".
 *  - Paladino (p82): "paladino do bem [...] sem deus específico".
 *
 * Stored alongside deus ids in the same slot so the picker has a single
 * source of truth; prereq checks treat them as "not devoto of a divindade"
 * (Arma Sagrada explicitly forbids `paladino-do-bem`, for instance).
 */
export const CULTO_PANTEAO = 'panteao'
export const CULTO_PALADINO_DO_BEM = 'paladino-do-bem'

export const DEUSES: Deus[] = [
  { id: 'aharadak', name: 'Aharadak', major: true, paladinoEligible: false, druidaEligible: false },
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

const CULTO_PANTEAO_OPTION: Deus = {
  id: CULTO_PANTEAO,
  name: 'Panteão',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}

const CULTO_PALADINO_DO_BEM_OPTION: Deus = {
  id: CULTO_PALADINO_DO_BEM,
  name: 'Paladino do Bem',
  major: false,
  paladinoEligible: false,
  druidaEligible: false,
}

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
 * in PDF Cap 3 (Religião):
 *  - Clérigo: any deus maior + 'Panteão' option (p57).
 *  - Paladino: 8 deuses whitelist + 'Paladino do Bem' option (p82).
 *  - Druida: Allihanna, Megalokk, Oceano (p61) — no non-divindade alt.
 */
export function devotoOptionsFor(className: string): Deus[] | null {
  switch (className) {
    case 'Clérigo':
      return [...DEUSES.filter((d) => d.major), CULTO_PANTEAO_OPTION]
    case 'Paladino':
      return [
        ...DEUSES.filter((d) => d.paladinoEligible),
        CULTO_PALADINO_DO_BEM_OPTION,
      ]
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
