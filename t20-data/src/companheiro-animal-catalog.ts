/**
 * Companheiro Animal — poder de Druida (PDF p61-62 sidebar).
 *
 * Catálogo dos 8 tipos-base disponíveis + amostras de animais + regras
 * dos augments (Aprimorado / Mágico / Lendário) + regras de morte e
 * reinvocação.
 *
 * Complementa:
 *  - `parceiro-rules.ts` — `companheiroAnimalTier(classLevel)` (7º → veterano, 15º → mestre)
 *  - `parceiro-benefits.ts` — bônus mecânico por (tipo × patamar)
 *  - `montaria-catalog.ts` — quando tipo = 'montaria', usa stats de p262
 *
 * Restrições da sidebar (p62):
 *  - Tipos base: 8 (ajudante/assassino/atirador/combatente/fortão/
 *    guardião/perseguidor/montaria).
 *  - Adepto/destruidor/magivocador/médico só desbloqueiam via
 *    Companheiro Animal Mágico (8º nível de druida).
 *  - Vigilante NÃO é tipo válido para animal em nenhum patamar.
 */

import type { ParceiroType } from './parceiro-rules'

/** Tipos de companheiro animal disponíveis no poder base (p62 sidebar). */
export type CompanheiroAnimalBaseTipo =
  | 'ajudante'
  | 'assassino'
  | 'atirador'
  | 'combatente'
  | 'fortao'
  | 'guardiao'
  | 'perseguidor'
  | 'montaria'

export const COMPANHEIRO_ANIMAL_BASE_TIPOS: readonly CompanheiroAnimalBaseTipo[] =
  Object.freeze([
    'ajudante',
    'assassino',
    'atirador',
    'combatente',
    'fortao',
    'guardiao',
    'perseguidor',
    'montaria',
  ])

/**
 * Tipos arcanos disponíveis exclusivamente via Companheiro Animal Mágico
 * (p62). Mapeiam para `ParceiroType`.
 */
export type CompanheiroAnimalMagicoTipo =
  | 'adepto'
  | 'destruidor'
  | 'magivocador'
  | 'medico'

export const COMPANHEIRO_ANIMAL_MAGICO_TIPOS: readonly CompanheiroAnimalMagicoTipo[] =
  Object.freeze(['adepto', 'destruidor', 'magivocador', 'medico'])

/** Vigilante nunca é tipo válido para companheiro animal (p62 sidebar). */
export const COMPANHEIRO_ANIMAL_VIGILANTE_EXCLUDED = true

/**
 * Amostras de animais por tipo (p62 sidebar). Combatente e Montaria
 * são listados como tipos válidos mas SEM bullets de amostra —
 * sidebar defere a p260 nas regras de parceiros.
 */
export const COMPANHEIRO_ANIMAL_SAMPLES: Readonly<
  Partial<Record<CompanheiroAnimalBaseTipo, readonly string[]>>
> = Object.freeze({
  ajudante: Object.freeze(['corvo', 'macaco', 'raposa', 'serpente']),
  assassino: Object.freeze(['lince', 'onça']),
  atirador: Object.freeze(['águia', 'falcão']),
  fortao: Object.freeze(['crocodilo', 'javali', 'leão', 'lobo']),
  guardiao: Object.freeze([
    'alce',
    'cão',
    'coruja',
    'tartaruga',
    'urso',
  ]),
  perseguidor: Object.freeze(['gambá', 'sabujo']),
})

/** Prosa verbatim das amostras (p62 sidebar). */
export const COMPANHEIRO_ANIMAL_SAMPLE_DESCRIPTIONS: Readonly<
  Partial<Record<CompanheiroAnimalBaseTipo, string>>
> = Object.freeze({
  ajudante: 'Corvo, macaco, raposa, serpente ou outro animal ágil ou esperto.',
  assassino: 'Lince, onça ou outro animal treinado para abater presas.',
  atirador:
    'Águia, falcão ou outro animal capaz de mergulhar rapidamente nos alvos de seus ataques à distância.',
  fortao:
    'Crocodilo, javali, leão, lobo ou outro animal capaz de lutar ao seu lado.',
  guardiao: 'Alce, cão, coruja, tartaruga, urso ou outro animal pesado ou atento.',
  perseguidor: 'Gambá, sabujo ou outro animal farejador.',
})

// ─── Regras de morte + reinvocação (p62 sidebar) ─────────────────────
/** Rodadas atordoado quando o companheiro morre. */
export const COMPANHEIRO_ANIMAL_STUN_ROUNDS_ON_DEATH = 1

/** Dias de prece/meditação para invocar novo companheiro após morte. */
export const COMPANHEIRO_ANIMAL_REINVOKE_DAYS = 1

// ─── Druida augments — níveis de destravamento (p62) ─────────────────
/** Nível de druida para "Companheiro Animal Aprimorado" (2º tipo). */
export const COMPANHEIRO_ANIMAL_APRIMORADO_LEVEL = 6

/** Nível de druida para "Companheiro Animal Mágico" (tipos arcanos). */
export const COMPANHEIRO_ANIMAL_MAGICO_LEVEL = 8

/** Nível de druida para "Companheiro Animal Lendário" (dobra bônus). */
export const COMPANHEIRO_ANIMAL_LENDARIO_LEVEL = 18

// ─── Pré-requisitos do poder base (p61-62) ───────────────────────────
export const COMPANHEIRO_ANIMAL_PREREQ_CARISMA = 1
export const COMPANHEIRO_ANIMAL_PREREQ_SKILL = 'Adestramento' as const

// ─── Type mapping ────────────────────────────────────────────────────
/**
 * Mapeia tipo base de companheiro animal para `ParceiroType` do módulo
 * de benefícios. `'montaria'` não tem correspondente em ParceiroType
 * (é uma categoria de parceiro em `montaria-catalog.ts`).
 */
export function baseTipoToParceiroType(
  tipo: CompanheiroAnimalBaseTipo,
): ParceiroType | 'montaria' {
  if (tipo === 'montaria') return 'montaria'
  return tipo
}

/** Verifica se um tipo é permitido no poder base (não requer Mágico). */
export function isBaseTipoAllowed(tipo: string): tipo is CompanheiroAnimalBaseTipo {
  return (COMPANHEIRO_ANIMAL_BASE_TIPOS as readonly string[]).includes(tipo)
}

/** Verifica se um tipo requer o poder "Companheiro Animal Mágico". */
export function requiresMagicoAugment(
  tipo: string,
): tipo is CompanheiroAnimalMagicoTipo {
  return (COMPANHEIRO_ANIMAL_MAGICO_TIPOS as readonly string[]).includes(tipo)
}
