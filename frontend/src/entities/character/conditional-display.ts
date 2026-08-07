import { resolveConditionalDisplay as tsResolveConditionalDisplay } from '@tormenta20/t20-data'
import {
  type ConditionalDisplayInput,
  type ConditionalDisplayRow,
  resolveConditionalDisplay as engineResolveConditionalDisplay,
} from '@/shared/lib/engine-wasm'

export type { ConditionalDisplayInput, ConditionalDisplayRow }

/**
 * Stance display resolution CHOKE POINT (Fase A.3): the non-stacking resolution
 * that keeps only winning conditional tiers (a Bárbaro 6's Fúria shows +3, not
 * the superseded +2). Same MODE-gate as the other engine choke points — the TS
 * branch (t20-data `resolveConditionalDisplay`) is TEST-ONLY and DCE'd from the
 * app bundle; production runs the Go/WASM `ResolveConditionalDisplay`. Pure
 * resolver, so no catalog priming is required, only the loaded engine (which
 * the stances panel already needs for its conditional list).
 */
export function resolveStanceDisplay(
  rows: ConditionalDisplayInput[],
): ConditionalDisplayRow[] {
  if (import.meta.env.MODE === 'test') {
    return tsResolveConditionalDisplay(rows)
  }
  return engineResolveConditionalDisplay(rows)
}
