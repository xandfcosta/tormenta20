/**
 * Perícias de Resistência (Fortitude, Reflexos, Vontade) — 3 perícias.
 *
 * PDF Cap 2 Perícias — sidebar "Perícias de Resistência" p119 verbatim:
 * "Fortitude, Reflexos e Vontade são usadas para resistir a efeitos
 * negativos, como uma explosão ou um encantamento de controle mental.
 * Por isso, são chamadas de perícias de resistência. Efeitos que afetem
 * seus 'testes de resistência' afetam todos os testes destas perícias.
 * Assim, um efeito que forneça +1 em testes de resistência fornece +1
 * em Fortitude, Reflexos e Vontade."
 *
 * FORTITUDE — CON (p119): resiste a efeitos que exigem vitalidade
 * (doenças, venenos); CD por efeito. Fôlego ao correr / sem respirar
 * = CD 15 +1 por teste anterior.
 *
 * REFLEXOS — DES (p122): resiste a efeitos que exigem reação rápida
 * (armadilhas, explosões); CD por efeito. Também usado para evitar
 * fintas (cross-ref [[enganacao-skill-usages]] uso Fintar).
 *
 * VONTADE — SAB (p123): resiste a efeitos que exigem determinação
 * (intimidação, encantamentos); CD por efeito. Testes de Vontade são
 * testes de resistência.
 *
 * Nota Tabela 2-1 p115: NENHUMA das 3 é somente treinada; NENHUMA
 * sofre penalidade de armadura. Cross-refs INTO estas perícias vindas
 * de [[intimidacao-skill-usages]], [[diplomacia-skill-usages]],
 * [[atuacao-skill-usages]], [[enganacao-skill-usages]].
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type SavingThrowSkill = 'fortitude' | 'reflexos' | 'vontade'

export type FortitudeUsageKind = 'resistir-efeito' | 'folego'
export type ReflexosUsageKind = 'resistir-efeito' | 'evitar-fintas'
export type VontadeUsageKind = 'resistir-efeito'

type FortitudeUsageCommon = {
  id: FortitudeUsageKind
  name: string
  effect: string
  bookPage: 119
}

type ReflexosUsageCommon = {
  id: ReflexosUsageKind
  name: string
  effect: string
  bookPage: 122
}

type VontadeUsageCommon = {
  id: VontadeUsageKind
  name: string
  effect: string
  bookPage: 123
}

export type FortitudeResistir = FortitudeUsageCommon & {
  kind: 'resistir-efeito'
  cdSource: 'per-effect'
  category: 'saving-throw'
}

export type FortitudeFolego = FortitudeUsageCommon & {
  kind: 'folego'
  cdBase: 15
  cdIncrementPerPreviousTest: 1
}

export type FortitudeUsage = FortitudeResistir | FortitudeFolego

export type ReflexosResistir = ReflexosUsageCommon & {
  kind: 'resistir-efeito'
  cdSource: 'per-effect'
  category: 'saving-throw'
}

export type ReflexosEvitarFintas = ReflexosUsageCommon & {
  kind: 'evitar-fintas'
  opposedBy: 'enganacao'
}

export type ReflexosUsage = ReflexosResistir | ReflexosEvitarFintas

export type VontadeResistir = VontadeUsageCommon & {
  kind: 'resistir-efeito'
  cdSource: 'per-effect'
  category: 'saving-throw'
}

export type VontadeUsage = VontadeResistir

// ─── Constantes ──────────────────────────────────────────────────────
/** Atributo-chave por perícia de resistência (Tabela 2-1 p115). */
export const SAVING_THROW_KEY_ATTRIBUTE: Readonly<
  Record<SavingThrowSkill, 'CON' | 'DES' | 'SAB'>
> = Object.freeze({
  fortitude: 'CON',
  reflexos: 'DES',
  vontade: 'SAB',
})

// Sidebar p119 verbatim
/** Bônus em "testes de resistência" aplica a todas as 3 perícias. */
export const SAVING_THROW_MODIFIER_APPLIES_TO_ALL = true

// Fortitude — Fôlego (p119 verbatim)
export const FORTITUDE_FOLEGO_CD_BASE = 15
export const FORTITUDE_FOLEGO_CD_INCREMENT = 1

// Flags Tabela 2-1 p115 — todos idênticos
export const FORTITUDE_TRAINED_ONLY = false
export const FORTITUDE_ARMOR_PENALTY = false
export const REFLEXOS_TRAINED_ONLY = false
export const REFLEXOS_ARMOR_PENALTY = false
export const VONTADE_TRAINED_ONLY = false
export const VONTADE_ARMOR_PENALTY = false

// ─── Catálogos ───────────────────────────────────────────────────────
export const FORTITUDE_USAGES: readonly FortitudeUsage[] = Object.freeze([
  {
    id: 'resistir-efeito',
    kind: 'resistir-efeito',
    name: 'Resistir a Efeito',
    cdSource: 'per-effect',
    category: 'saving-throw',
    effect:
      'Resiste a efeitos que exigem vitalidade (doenças, venenos); CD determinada pelo efeito.',
    bookPage: 119,
  },
  {
    id: 'folego',
    kind: 'folego',
    name: 'Fôlego',
    cdBase: 15,
    cdIncrementPerPreviousTest: 1,
    effect:
      'Mantém fôlego correndo ou sem respirar; CD 15 +1 por teste anterior (cumulativo).',
    bookPage: 119,
  },
])

export const REFLEXOS_USAGES: readonly ReflexosUsage[] = Object.freeze([
  {
    id: 'resistir-efeito',
    kind: 'resistir-efeito',
    name: 'Resistir a Efeito',
    cdSource: 'per-effect',
    category: 'saving-throw',
    effect:
      'Resiste a efeitos que exigem reação rápida (armadilhas, explosões); CD determinada pelo efeito.',
    bookPage: 122,
  },
  {
    id: 'evitar-fintas',
    kind: 'evitar-fintas',
    opposedBy: 'enganacao',
    name: 'Evitar Fintas',
    effect:
      'Oposto ao teste de Enganação/Fintar do adversário em alcance curto.',
    bookPage: 122,
  },
])

export const VONTADE_USAGES: readonly VontadeUsage[] = Object.freeze([
  {
    id: 'resistir-efeito',
    kind: 'resistir-efeito',
    name: 'Resistir a Efeito',
    cdSource: 'per-effect',
    category: 'saving-throw',
    effect:
      'Resiste a efeitos que exigem determinação (intimidação, encantamentos); CD determinada pelo efeito. Testes de Vontade são testes de resistência.',
    bookPage: 123,
  },
])

export const fortitudeUsageByKind = makeUsageByKind<FortitudeUsageKind, FortitudeUsage>(
  FORTITUDE_USAGES,
  'fortitudeUsageByKind',
)

export const reflexosUsageByKind = makeUsageByKind<ReflexosUsageKind, ReflexosUsage>(
  REFLEXOS_USAGES,
  'reflexosUsageByKind',
)

export const vontadeUsageByKind = makeUsageByKind<VontadeUsageKind, VontadeUsage>(
  VONTADE_USAGES,
  'vontadeUsageByKind',
)

// ─── Helpers — Fortitude Fôlego ─────────────────────────────────────
/** CD de Fortitude para fôlego escalando por testes anteriores. */
export function folegoCd(previousTests: number): number {
  return (
    FORTITUDE_FOLEGO_CD_BASE + previousTests * FORTITUDE_FOLEGO_CD_INCREMENT
  )
}

// ─── Helpers — comum ────────────────────────────────────────────────
/**
 * Aplica bônus/penalidade em "testes de resistência" a uma perícia
 * de resistência específica — os três compartilham modificadores
 * genéricos por regra da sidebar p119.
 */
export function savingThrowModifierAppliesTo(_skill: SavingThrowSkill): boolean {
  return SAVING_THROW_MODIFIER_APPLIES_TO_ALL
}

/** Atributo-chave para a perícia de resistência. */
export function savingThrowKeyAttribute(
  skill: SavingThrowSkill,
): 'CON' | 'DES' | 'SAB' {
  return SAVING_THROW_KEY_ATTRIBUTE[skill]
}
