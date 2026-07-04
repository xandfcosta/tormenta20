/**
 * Perícia Iniciativa (DES, aberta, sem penalidade de armadura) — 1 uso.
 *
 * PDF Cap 2 Perícias — Iniciativa (livro p119). Header verbatim:
 * "INICIATIVA — DES".
 * Corpo verbatim: "Esta perícia determina sua velocidade de reação em
 * situações de perigo. Quando uma cena do jogo começa, cada personagem
 * envolvido faz um teste de Iniciativa. Eles então agem em ordem
 * decrescente dos resultados."
 *
 * Único uso:
 *  - Teste de Iniciativa (p119) — rolado por cada personagem no início
 *    da cena; personagens agem em ordem decrescente dos resultados.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO sofre penalidade de
 * armadura. Cross-refs INTO Iniciativa (do outro lado): Acrobacia/Passar
 * por Inimigo usa Iniciativa como uma opção do "melhor" do oponente;
 * Guerra/Plano de Ação concede +5 na Iniciativa de um aliado.
 */

// ─── Types ────────────────────────────────────────────────────────────
export type IniciativaUsageKind = 'teste-iniciativa'

type UsageCommon = {
  id: IniciativaUsageKind
  name: string
  effect: string
  bookPage: 119
}

export type IniciativaTeste = UsageCommon & {
  kind: 'teste-iniciativa'
  /** Rolado ao início de uma cena de perigo. */
  rolledAtSceneStart: true
  /** Ação em ordem decrescente dos resultados. */
  orderDescending: true
}

export type IniciativaUsage = IniciativaTeste

// ─── Constantes ──────────────────────────────────────────────────────
// Flags Tabela 2-1 p115
export const INICIATIVA_TRAINED_ONLY = false
export const INICIATIVA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const INICIATIVA_USAGES: readonly IniciativaUsage[] = Object.freeze([
  {
    id: 'teste-iniciativa',
    kind: 'teste-iniciativa',
    name: 'Teste de Iniciativa',
    rolledAtSceneStart: true,
    orderDescending: true,
    effect:
      'Rolado no início da cena; personagens agem em ordem decrescente dos resultados.',
    bookPage: 119,
  },
])

const usagesByKind = new Map<IniciativaUsageKind, IniciativaUsage>(
  INICIATIVA_USAGES.map((u) => [u.kind, u]),
)

export function iniciativaUsageByKind(
  kind: IniciativaUsageKind,
): IniciativaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`iniciativaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers ────────────────────────────────────────────────────────
/** Comparador para ordenação DESCENDENTE por Iniciativa. */
export function compareIniciativaDesc(a: number, b: number): number {
  return b - a
}

/**
 * Retorna cópia dos participantes ordenada por Iniciativa (decrescente).
 * Empates preservam a ordem original (stable sort). Não modifica o input.
 */
export function sortByIniciativaDesc<T extends { iniciativa: number }>(
  participants: readonly T[],
): T[] {
  return [...participants].sort((a, b) => compareIniciativaDesc(a.iniciativa, b.iniciativa))
}
