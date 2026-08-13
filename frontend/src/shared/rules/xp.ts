/**
 * XP & advancement — PDF Cap 8 "Pontos de Experiência" (book p326-332).
 *
 * T20 does NOT publish a cumulative "XP to reach level N" table the way
 * D&D does. Instead, leveling is implicit in the encounter formula:
 *
 *   XP por desafio = ND × 1.000, dividido entre o grupo.
 *
 * For a four-character party, the math collapses to "~`nível` × 1.000
 * XP per character to level up" (book p332 sidebar: "um grupo de quatro
 * personagens deve vencer quatro ameaças de ND igual ao seu nível para
 * subir para o próximo").
 *
 * Encoded here:
 *   - encounter XP formula (`encounterXp`)
 *   - outcome multiplier (full / draw / loss) per p326
 *   - "irrelevant challenge" gate (ND 5+ below party level = 0 XP)
 *   - per-character XP threshold for the next level
 *   - level-up benefits enumeration (PV/PM/poder/perícias) per Treinamento p277
 */
export const MAX_LEVEL = 20

export type EncounterOutcome = 'win' | 'draw' | 'loss'

export const OUTCOME_MULTIPLIER: Record<EncounterOutcome, number> = {
  win: 1,
  draw: 0.5,
  loss: 0.25,
}

/** Lowest ND - partyLevel gap below which an encounter awards 0 XP. */
export const IRRELEVANT_ND_GAP = 5

/**
 * XP awarded to each character for an encounter.
 *
 *   base = ND × 1.000 × outcome multiplier
 *   per char = base / partySize
 *
 * Returns 0 when:
 *   - `partySize ≤ 0` (nobody to award)
 *   - `nd ≤ partyLevel - 5` (PDF: "Desafios Irrelevantes")
 *   - `nd ≤ 0`
 */
export function encounterXp(args: {
  nd: number
  partyLevel: number
  partySize: number
  outcome: EncounterOutcome
}): number {
  const { nd, partyLevel, partySize, outcome } = args
  if (partySize <= 0 || nd <= 0) return 0
  if (nd <= partyLevel - IRRELEVANT_ND_GAP) return 0
  const total = nd * 1000 * OUTCOME_MULTIPLIER[outcome]
  return Math.floor(total / partySize)
}

/**
 * Cumulative XP required to reach the given character level. Uses the
 * book-side approximation: "four ND=level challenges per level". For a
 * party of four with full wins, that comes to `level × 1.000` per
 * character per level — i.e. `1.000 + 2.000 + … + (N-1) × 1.000`.
 *
 * Returns 0 for level ≤ 1 (no XP needed to start at L1).
 * Clamps to MAX_LEVEL (no published progression beyond L20).
 */
export function xpToReachLevel(level: number): number {
  if (level <= 1) return 0
  const clamped = Math.min(level, MAX_LEVEL)
  // Sum 1 + 2 + … + (clamped - 1) all × 1000.
  const n = clamped - 1
  return ((n * (n + 1)) / 2) * 1000
}

/**
 * Highest level the character has earned with `xp` total experience.
 * Inverse of `xpToReachLevel`. Returns 1 for any xp < 1000, clamps to
 * MAX_LEVEL.
 */
export function levelForXp(xp: number): number {
  if (xp < 1000) return 1
  for (let level = MAX_LEVEL; level >= 2; level--) {
    if (xp >= xpToReachLevel(level)) return level
  }
  return 1
}

/**
 * Benefits granted by advancing to the given level. Per Treinamento
 * (book p277), the closest enumeration the rulebook gives:
 *
 *   - PV equivalentes ao próximo nível
 *   - PM equivalentes ao próximo nível
 *   - Uma habilidade de classe do próximo nível
 *   - +1 em todas as perícias treinadas (somente quando o próximo nível é par)
 *
 * Per-class power slot mapping lives in `class-vitals.ts` /
 * `abilities/classes/index.ts`; this enum is the *generic* benefit list.
 */
export type LevelUpBenefit =
  | 'pv-next-level'
  | 'pm-next-level'
  | 'class-power-slot'
  | 'expertise-plus-one'

export function levelUpBenefits(nextLevel: number): LevelUpBenefit[] {
  if (nextLevel < 2 || nextLevel > MAX_LEVEL) return []
  const out: LevelUpBenefit[] = [
    'pv-next-level',
    'pm-next-level',
    'class-power-slot',
  ]
  if (nextLevel % 2 === 0) out.push('expertise-plus-one')
  return out
}
