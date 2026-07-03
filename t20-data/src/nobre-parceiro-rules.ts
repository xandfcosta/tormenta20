/**
 * Nobre — poderes que geram parceiros (PDF p79-80).
 *
 * Autoridade Feudal (p79) e Título (p80) são as únicas entradas do
 * Nobre que geram parceiros. Estrategista/General/Palavras Afiadas
 * direcionam "aliados em alcance curto" — não usam o subsistema de
 * parceiros.
 *
 * Deltas vs `cavaleiro-parceiro-rules.ts`:
 *  - Autoridade Feudal é texto-idêntico ao Cavaleiro (mesmo L6,
 *    mesma tier iniciante, mesma duração "até o fim da aventura").
 *  - Título do Nobre **não tem pré-requisito de nível** nem exige
 *    Autoridade Feudal, terras ou suserano (assimetria confirmada
 *    na p80 vs Cavaleiro Título p55).
 *
 * Cross-ref: `parceiro-rules.ts:parceiroLimit(pcLevel)` — todos contam.
 */

import type { ParceiroTier } from './parceiro-rules'

export type NobreParceiroPowerId = 'autoridade-feudal' | 'titulo'

/** Duração do parceiro concedido. */
export type ParceiroDuration = 'ate-fim-aventura' | 'inicio-aventura' | 'dia'

export type NobreParceiroPower = {
  id: NobreParceiroPowerId
  name: string
  bookPage: 79 | 80
  minLevel: number
  countsAgainstLimit: boolean
  grantedTier: ParceiroTier
  duration: ParceiroDuration
  additionalPrereqs: readonly string[]
  /** Custo em PM para invocar (Autoridade Feudal = 2). null = sem custo. */
  pmCost: number | null
}

export const NOBRE_AUTORIDADE_FEUDAL_MIN_LEVEL = 6

/**
 * Título do Nobre não lista pré-requisito de nível no PDF (p80). Nobre
 * base tem 3 poderes iniciais + 1 eletivo por nível ímpar. Encodo
 * `minLevel = 1` com nota — assimetria confirmada vs Cavaleiro Título.
 */
export const NOBRE_TITULO_MIN_LEVEL = 1

const RAW: readonly NobreParceiroPower[] = [
  {
    id: 'autoridade-feudal',
    name: 'Autoridade Feudal',
    bookPage: 79,
    minLevel: NOBRE_AUTORIDADE_FEUDAL_MIN_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'iniciante',
    duration: 'ate-fim-aventura',
    additionalPrereqs: ['local onde sua posição carregue influência'],
    pmCost: 2,
  },
  {
    id: 'titulo',
    name: 'Título',
    bookPage: 80,
    minLevel: NOBRE_TITULO_MIN_LEVEL,
    countsAgainstLimit: true,
    grantedTier: 'veterano',
    duration: 'inicio-aventura',
    additionalPrereqs: [],
    pmCost: null,
  },
]

export const NOBRE_PARCEIRO_POWERS: readonly NobreParceiroPower[] =
  Object.freeze(RAW)

export function nobreParceiroPowerById(
  id: NobreParceiroPowerId,
): NobreParceiroPower | undefined {
  return NOBRE_PARCEIRO_POWERS.find((p) => p.id === id)
}

/** Filtra poderes disponíveis ao nível informado. */
export function unlockedNobreParceiroPowers(
  nobreLevel: number,
): readonly NobreParceiroPower[] {
  if (nobreLevel < 1) {
    throw new Error(
      `unlockedNobreParceiroPowers: nobreLevel must be ≥ 1, got ${nobreLevel}`,
    )
  }
  return NOBRE_PARCEIRO_POWERS.filter((p) => p.minLevel <= nobreLevel)
}
