/**
 * Perícia Adestramento (CAR, treinada) — 2 usos canônicos.
 *
 * PDF Cap 2 Perícias — Adestramento (livro p115).
 *
 * Header verbatim (Tabela 2-1 + entrada): "ADESTRAMENTO — CAR, treinada,
 * sem penalidade de armadura". Intro verbatim: "Você sabe lidar com animais."
 *
 * Diferente de D&D 3.5, T20 core simplifica a perícia: SÓ existem
 * Acalmar Animal e Manejar Animal. NÃO existem sub-usos "Treinar Animal",
 * "Ensinar Truque", "Criar Filhotes", nem lista padronizada de truques.
 * Exemplos de comandos citados inline em Manejar Animal:
 * "atacar", "sentar", "vigiar"…
 *
 * Cross-ref:
 *  - `montaria-catalog.ts` — cavalos/montarias; Manejar Animal substitui
 *    Pilotagem para veículos de tração animal (verbatim).
 *  - `companheiro-animal-catalog.ts` / `parceiro-rules.ts` — parceiros
 *    animais têm regras próprias que não passam por Adestramento.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type AdestramentoUsageKind = 'acalmar-animal' | 'manejar-animal'

/** Ação exigida por cada uso. */
export type AdestramentoAction = 'completa' | 'movimento'

type UsageCommon = {
  id: AdestramentoUsageKind
  name: string
  action: AdestramentoAction
  dc: number
  effect: string
  bookPage: 115
}

export type AdestramentoAcalmarAnimal = UsageCommon & {
  kind: 'acalmar-animal'
  action: 'completa'
  dc: 25
  /** Aplica-se a qualquer animal nervoso/agressivo, treinado ou não. */
  requiresTrainedTarget: false
}

export type AdestramentoManejarAnimal = UsageCommon & {
  kind: 'manejar-animal'
  action: 'movimento'
  dc: 15
  /** Verbatim: "para a qual foi treinado". Só funciona em tarefa treinada. */
  requiresTrainedTarget: true
  /** Verbatim: "usar Adestramento como Pilotagem para veículos de tração animal". */
  substitutesPilotagemForAnimalDrawnVehicles: true
}

export type AdestramentoUsage =
  | AdestramentoAcalmarAnimal
  | AdestramentoManejarAnimal

// ─── Constantes ──────────────────────────────────────────────────────
/** CD verbatim p115. */
export const ACALMAR_ANIMAL_CD = 25
/** CD verbatim p115. */
export const MANEJAR_ANIMAL_CD = 15

/**
 * Exemplos de comandos citados inline (não é lista fechada).
 * Verbatim p115: "atacar", "sentar", "vigiar"…
 */
export const MANEJAR_ANIMAL_EXAMPLE_COMMANDS: readonly string[] = Object.freeze([
  'atacar',
  'sentar',
  'vigiar',
])

/** Perícia é apenas treinada (Tabela 2-1). */
export const ADESTRAMENTO_TRAINED_ONLY = true

/** Sem penalidade de armadura (Tabela 2-1). */
export const ADESTRAMENTO_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const ADESTRAMENTO_USAGES: readonly AdestramentoUsage[] = Object.freeze([
  {
    id: 'acalmar-animal',
    kind: 'acalmar-animal',
    name: 'Acalmar Animal',
    action: 'completa',
    dc: ACALMAR_ANIMAL_CD,
    requiresTrainedTarget: false,
    effect:
      'Acalma animal nervoso ou agressivo (controlar cavalo assustado, convencer lobo a não atacar).',
    bookPage: 115,
  },
  {
    id: 'manejar-animal',
    kind: 'manejar-animal',
    name: 'Manejar Animal',
    action: 'movimento',
    dc: MANEJAR_ANIMAL_CD,
    requiresTrainedTarget: true,
    substitutesPilotagemForAnimalDrawnVehicles: true,
    effect:
      'Faz animal realizar tarefa para a qual foi treinado; substitui Pilotagem em veículos de tração animal.',
    bookPage: 115,
  },
])

export const adestramentoUsageByKind = makeUsageByKind<AdestramentoUsageKind, AdestramentoUsage>(
  ADESTRAMENTO_USAGES,
  'adestramentoUsageByKind',
)

// ─── Helpers ─────────────────────────────────────────────────────────
/** CD de Acalmar Animal — fixa 25 (verbatim). */
export function acalmarAnimalCd(): number {
  return ACALMAR_ANIMAL_CD
}

/** CD de Manejar Animal — fixa 15 (verbatim). */
export function manejarAnimalCd(): number {
  return MANEJAR_ANIMAL_CD
}

/**
 * Verbatim p115: "Isso permite usar Adestramento como Pilotagem para
 * veículos de tração animal." Retorna true se o veículo em questão for
 * de tração animal.
 */
export function canSubstitutePilotagemForVehicle(
  vehicleIsAnimalDrawn: boolean,
): boolean {
  return vehicleIsAnimalDrawn
}
