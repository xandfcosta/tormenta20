/**
 * Perícia Pontaria (DES, aberta, sem penalidade de armadura) — 1 uso.
 *
 * PDF Cap 2 Perícias — Pontaria (livro p122). Header verbatim:
 * "PONTARIA — DES".
 * Corpo verbatim: "Você usa Pontaria para fazer ataques à distância.
 * A CD é a Defesa do alvo. Se você acertar, causa dano de acordo com
 * a arma utilizada. Veja o Capítulo 5: Jogando para as regras completas
 * de ataque."
 *
 * Único uso:
 *  - Ataque à Distância (p122) — CD = Defesa do alvo; sucesso = dano
 *    conforme arma. Alcance, cobertura, mirar etc. vivem no Cap 5.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO sofre penalidade de
 * armadura. Espelha [[luta-skill-usages]] com foco em ataques à distância.
 */

import { makeUsageByKind } from './skill-usage-resolver'

// ─── Types ────────────────────────────────────────────────────────────
export type PontariaUsageKind = 'ataque-a-distancia'

type UsageCommon = {
  id: PontariaUsageKind
  name: string
  effect: string
  bookPage: 122
}

export type PontariaAtaqueADistancia = UsageCommon & {
  kind: 'ataque-a-distancia'
  cdEqualsTargetDefesa: true
  damageBy: 'arma-utilizada'
  crossRef: 'capitulo-5-jogando'
}

export type PontariaUsage = PontariaAtaqueADistancia

// ─── Constantes ──────────────────────────────────────────────────────
// Flags Tabela 2-1 p115
export const PONTARIA_TRAINED_ONLY = false
export const PONTARIA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const PONTARIA_USAGES: readonly PontariaUsage[] = Object.freeze([
  {
    id: 'ataque-a-distancia',
    kind: 'ataque-a-distancia',
    name: 'Ataque à Distância',
    cdEqualsTargetDefesa: true,
    damageBy: 'arma-utilizada',
    crossRef: 'capitulo-5-jogando',
    effect:
      'Ataque à distância; CD = Defesa do alvo; sucesso causa dano da arma utilizada. Regras completas no Cap 5.',
    bookPage: 122,
  },
])

export const pontariaUsageByKind = makeUsageByKind<PontariaUsageKind, PontariaUsage>(
  PONTARIA_USAGES,
  'pontariaUsageByKind',
)

// ─── Helpers — Ataque à Distância ───────────────────────────────────
/** CD do ataque à distância = Defesa do alvo (verbatim p122). */
export function ataqueADistanciaCd(alvoDefesa: number): number {
  return alvoDefesa
}

export type AtaqueADistanciaOutcome = 'hit' | 'miss'

/** Resolve o ataque (empate favorece atacante — convenção T20). */
export function ataqueADistanciaOutcome(
  ataqueTotal: number,
  alvoDefesa: number,
): AtaqueADistanciaOutcome {
  return ataqueTotal >= alvoDefesa ? 'hit' : 'miss'
}
