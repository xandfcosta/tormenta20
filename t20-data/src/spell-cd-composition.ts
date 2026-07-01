/**
 * Spell save CD composition — junta base CD com todos os bumps
 * cumulativos: Especialista em Escola + Magia Pungente + Mitral
 * esotérico + Poderoso enchant + Fortalecimento Arcano.
 *
 * PDF refs:
 *  - Base CD: p171 — `spellSaveDc(casterLevel, keyMod)` já em `spells.ts`
 *  - Especialista em Escola (p38 Arcanista): +2 CD escola escolhida
 *  - Magia Pungente (p38 Arcanista elective): +2 CD ao pagar +1 PM
 *  - Mitral esotérico (p167 item): +2 CD ao pagar +2 PM
 *  - Poderoso enchant (esotérico Tabela 8-8 superior; p165): +1 CD static
 *  - Fortalecimento Arcano (p38 Arcanista, nível 5+): +1 CD; sobe pra
 *    +2 se pode lançar magias de 4º círculo
 *
 * NOTA: PM extra dos opt-ins (Magia Pungente/Mitral) NÃO é aplicado
 * aqui — caller adiciona ao `augmentPm` de `totalSpellPmCost`. Este
 * módulo é CD-only.
 */
import type { SpellSchool } from './spells'
import {
  spellSaveCdWithSpecialization,
  type SchoolSpecialization,
} from './school-specialization'

/** Poderoso enchant: +1 CD por item equipado (raro empilhar). */
export const PODEROSO_CD_BONUS = 1

/** Fortalecimento Arcano nível 1-4: +1 CD. */
export const FORTALECIMENTO_BASIC_BONUS = 1

/** Fortalecimento Arcano com 4º círculo desbloqueado: +2 CD. */
export const FORTALECIMENTO_ADVANCED_BONUS = 2

/** Magia Pungente (arcanista elective): +2 CD por 1 PM. */
export const MAGIA_PUNGENTE_CD_BONUS = 2
export const MAGIA_PUNGENTE_PM_EXTRA = 1

/** Mitral esotérico (item): +2 CD por 2 PM. */
export const MITRAL_ESOTERICO_CD_BONUS = 2
export const MITRAL_ESOTERICO_PM_EXTRA = 2

export type FortalecimentoLevel = 'none' | 'basic' | 'advanced'

export type SpellCdComposition = {
  baseCd: number
  spellSchool: SpellSchool
  specializations?: readonly SchoolSpecialization[]
  /** Número de itens Poderoso equipados (raro > 1 mas possível). */
  poderosoStacks?: number
  fortalecimentoArcano?: FortalecimentoLevel
  /** Opt-in por cast: paga +1 PM extra. */
  magiaPungentePaid?: boolean
  /** Opt-in por cast: paga +2 PM extra. */
  mitralEsotericoPaid?: boolean
}

function fortalecimentoBonus(level: FortalecimentoLevel): number {
  if (level === 'basic') return FORTALECIMENTO_BASIC_BONUS
  if (level === 'advanced') return FORTALECIMENTO_ADVANCED_BONUS
  return 0
}

/**
 * CD final do teste de resistência a uma magia após todos os bumps
 * empilhados. Nunca negativo (embora nenhum bônus subtrai — apenas
 * defensivo).
 */
export function totalSpellSaveCd(c: SpellCdComposition): number {
  if (c.poderosoStacks !== undefined && c.poderosoStacks < 0) {
    throw new Error(
      `totalSpellSaveCd: poderosoStacks must be ≥ 0, got ${c.poderosoStacks}`,
    )
  }
  let cd = spellSaveCdWithSpecialization(
    c.baseCd,
    c.spellSchool,
    c.specializations ?? [],
  )
  cd += (c.poderosoStacks ?? 0) * PODEROSO_CD_BONUS
  cd += fortalecimentoBonus(c.fortalecimentoArcano ?? 'none')
  if (c.magiaPungentePaid) cd += MAGIA_PUNGENTE_CD_BONUS
  if (c.mitralEsotericoPaid) cd += MITRAL_ESOTERICO_CD_BONUS
  return cd
}

/**
 * PM extra que deve ser somado ao custo total da magia por causa
 * dos opt-ins de CD. Caller passa esse valor pra `augmentPm` de
 * `totalSpellPmCost`.
 */
export function extraPmFromCdBoosts(
  magiaPungentePaid?: boolean,
  mitralEsotericoPaid?: boolean,
): number {
  let extra = 0
  if (magiaPungentePaid) extra += MAGIA_PUNGENTE_PM_EXTRA
  if (mitralEsotericoPaid) extra += MITRAL_ESOTERICO_PM_EXTRA
  return extra
}
