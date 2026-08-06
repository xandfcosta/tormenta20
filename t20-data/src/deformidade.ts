/**
 * Deformidade — habilidade racial do Lefou (book p23).
 *
 * "Você recebe +2 em duas perícias a sua escolha. Cada um desses bônus conta
 * como um poder da Tormenta (mas você não perde Carisma por eles). Você pode
 * trocar um desses bônus por um poder da Tormenta a sua escolha."
 *
 * Encoded rules:
 *  - 2 slots; each unswapped slot = +2 in a chosen perícia.
 *  - At most ONE slot may become a real poder da Tormenta (single
 *    `tormentaPower` field enforces the book's "um desses bônus").
 *  - Perícia bonuses count as poderes da Tormenta for prerequisites, but do
 *    NOT cost Carisma; only the swapped real power does (p136 loss rule).
 *  - Under-filling is allowed (homebrew: GM may grant fewer picks).
 */
import type { ExpertiseName } from './expertises'
import { EXPERTISE_NAMES } from './expertises'
// raceWithDeformidade reads RACES_CATALOG — moved to ./deformidade-race so this
// module tree-shakes free of the abilities catalog in the front (B.3).
export { raceWithDeformidade } from './deformidade-race'
import type { SkillId } from './skill-index'
import { SKILL_IDS } from './skill-index'
import type { TormentaPower, TormentaPowerId } from './tormenta'

export type DeformidadeChoice = {
  /** Perícias que recebem +2 (um slot cada). */
  pericias: readonly ExpertiseName[]
  /** Slot trocado por um poder da Tormenta real — no máximo um (p23). */
  tormentaPower?: TormentaPowerId
}

export const DEFORMIDADE_SLOTS = 2
export const DEFORMIDADE_PERICIA_BONUS = 2

/** Slots consumed by the choice (perícias + the optional swapped power). */
export function deformidadeSlotsUsed(choice: DeformidadeChoice): number {
  return choice.pericias.length + (choice.tormentaPower ? 1 : 0)
}

/**
 * Validate a choice; returns human-readable warnings (empty = valid).
 * Under-filled choices are valid — only over-fills / duplicates / unknown
 * ids warn. Example: `validateDeformidade({ pericias: ['Furtividade'] })`.
 */
export function validateDeformidade(
  choice: DeformidadeChoice,
  tormentaPowers: Readonly<Record<string, TormentaPower>>,
): string[] {
  const warnings: string[] = []
  const used = deformidadeSlotsUsed(choice)
  if (used > DEFORMIDADE_SLOTS) {
    warnings.push(
      `Deformidade: ${used} escolhas excedem os ${DEFORMIDADE_SLOTS} slots (perícias=${choice.pericias.length}, poder=${choice.tormentaPower ?? 'nenhum'})`,
    )
  }
  if (new Set(choice.pericias).size !== choice.pericias.length) {
    warnings.push(
      `Deformidade: perícias duplicadas em [${choice.pericias.join(', ')}]`,
    )
  }
  for (const name of choice.pericias) {
    if (!(EXPERTISE_NAMES as readonly string[]).includes(name)) {
      warnings.push(`Deformidade: perícia desconhecida "${name}"`)
    }
  }
  if (choice.tormentaPower && !(choice.tormentaPower in tormentaPowers)) {
    warnings.push(
      `Deformidade: poder da Tormenta desconhecido "${choice.tormentaPower}"`,
    )
  }
  return warnings
}

/**
 * How many "poderes da Tormenta" this choice counts as for PREREQUISITES —
 * perícia bonuses count (p23), so 2 perícias = 2 powers. Carisma loss is a
 * different count: only the real `tormentaPower` (see p136 rule).
 */
export function deformidadeTormentaPowerCount(choice: DeformidadeChoice): number {
  return deformidadeSlotsUsed(choice)
}

/**
 * Powers pickable as the Deformidade swap, given how many perícia bonuses are
 * already placed (they count as owned powers for `requiresOtherPowers`).
 * `requiresPower`-gated entries are never available — a specific power can't
 * be satisfied by perícia bonuses.
 */
export function deformidadeAvailablePowers(
  tormentaPowers: Readonly<Record<string, TormentaPower>>,
  periciaCount: number,
): TormentaPower[] {
  return Object.values(tormentaPowers).filter(
    (p) => !p.requiresPower && p.requiresOtherPowers <= periciaCount,
  )
}

/** PT expertise name → SkillId slug (NFD-strip + lowercase). */
export function expertiseNameToSkillId(name: string): SkillId | undefined {
  const normalized = name
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .trim()
  return (SKILL_IDS as readonly string[]).includes(normalized)
    ? (normalized as SkillId)
    : undefined
}

/** Chosen perícias as SkillIds; unknown names become warnings, not throws. */
export function deformidadeSkillIds(
  choice: DeformidadeChoice,
  warnings: string[],
): SkillId[] {
  const out: SkillId[] = []
  for (const name of choice.pericias) {
    const id = expertiseNameToSkillId(name)
    if (!id) {
      warnings.push(`Deformidade: perícia sem SkillId correspondente "${name}"`)
      continue
    }
    out.push(id)
  }
  return out
}
