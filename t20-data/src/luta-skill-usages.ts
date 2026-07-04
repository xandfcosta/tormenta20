/**
 * Perícia Luta (FOR, aberta, sem penalidade de armadura) — 1 uso.
 *
 * PDF Cap 2 Perícias — Luta (livro p121). Header verbatim:
 * "LUTA — FOR".
 * Corpo verbatim: "Você usa Luta para fazer ataques corpo a corpo.
 * A CD é a Defesa do alvo. Se você acertar, causa dano de acordo com
 * a arma utilizada. Veja o Capítulo 5: Jogando para as regras completas
 * de combate."
 *
 * Único uso:
 *  - Ataque Corpo a Corpo (p121) — CD = Defesa do alvo; sucesso = dano
 *    conforme arma.
 *
 * Nota Tabela 2-1 p115: NÃO é somente treinada; NÃO sofre penalidade de
 * armadura. Manobras (agarrar, derrubar, desarmar) vivem no Cap 5 e não
 * são usos de perícia — veja [[maneuvers]] / [[combat-actions]].
 */

// ─── Types ────────────────────────────────────────────────────────────
export type LutaUsageKind = 'ataque-corpo-a-corpo'

type UsageCommon = {
  id: LutaUsageKind
  name: string
  effect: string
  bookPage: 121
}

export type LutaAtaqueCorpoACorpo = UsageCommon & {
  kind: 'ataque-corpo-a-corpo'
  cdEqualsTargetDefesa: true
  damageBy: 'arma-utilizada'
  crossRef: 'capitulo-5-jogando'
}

export type LutaUsage = LutaAtaqueCorpoACorpo

// ─── Constantes ──────────────────────────────────────────────────────
// Flags Tabela 2-1 p115
export const LUTA_TRAINED_ONLY = false
export const LUTA_ARMOR_PENALTY = false

// ─── Catálogo ─────────────────────────────────────────────────────────
export const LUTA_USAGES: readonly LutaUsage[] = Object.freeze([
  {
    id: 'ataque-corpo-a-corpo',
    kind: 'ataque-corpo-a-corpo',
    name: 'Ataque Corpo a Corpo',
    cdEqualsTargetDefesa: true,
    damageBy: 'arma-utilizada',
    crossRef: 'capitulo-5-jogando',
    effect:
      'Ataque corpo a corpo; CD = Defesa do alvo; sucesso causa dano da arma utilizada. Regras completas no Cap 5.',
    bookPage: 121,
  },
])

const usagesByKind = new Map<LutaUsageKind, LutaUsage>(
  LUTA_USAGES.map((u) => [u.kind, u]),
)

export function lutaUsageByKind(kind: LutaUsageKind): LutaUsage {
  const usage = usagesByKind.get(kind)
  if (!usage) {
    throw new Error(`lutaUsageByKind: unknown kind ${kind}`)
  }
  return usage
}

// ─── Helpers — Ataque Corpo a Corpo ─────────────────────────────────
/** CD do ataque = Defesa do alvo (verbatim p121). */
export function ataqueCorpoACorpoCd(alvoDefesa: number): number {
  return alvoDefesa
}

export type AtaqueOutcome = 'hit' | 'miss'

/** Resolve o ataque (empate favorece atacante — convenção T20). */
export function ataqueCorpoACorpoOutcome(
  ataqueTotal: number,
  alvoDefesa: number,
): AtaqueOutcome {
  return ataqueTotal >= alvoDefesa ? 'hit' : 'miss'
}
