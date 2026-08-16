import {
  type ConditionalDisplayInput,
  type ConditionalDisplayRow,
  resolveConditionalDisplay as engineResolveConditionalDisplay,
} from '@/shared/lib/engine-wasm'

export type { ConditionalDisplayInput, ConditionalDisplayRow }

/**
 * Stance display resolution CHOKE POINT: the non-stacking resolution that keeps
 * only winning conditional tiers — a Bárbaro 6's Fúria shows +3, not the
 * superseded +2 as well.
 *
 * Quem resolve é o Go/WASM `ResolveConditionalDisplay`, em todos os ambientes —
 * o ramo TS do `t20-data` não existe mais (ALE-104), e o vitest carrega o mesmo
 * `.wasm` da produção. Pure
 * resolver, so it needs no catalog priming — only the loaded engine, which the
 * stance list already depends on for its conditionals.
 *
 * @example resolveStanceDisplay(group.entries.map(toDisplayInput))
 */
export function resolveStanceDisplay(
  rows: ConditionalDisplayInput[],
): ConditionalDisplayRow[] {
  return engineResolveConditionalDisplay(rows)
}
