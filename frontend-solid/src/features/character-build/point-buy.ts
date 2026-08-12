import {
  type AttributeKey,
  pointBuySpent,
  pointBuyWarnings,
} from '@tormenta20/t20-data'
import {
  pointBuyStatus as enginePointBuyStatus,
  type PointBuyStatus,
} from '@/shared/lib/engine-wasm'

export type { PointBuyStatus }

/**
 * Creation point-buy status (p17) CHOKE POINT (migração TS→Go): total spent +
 * advisory warnings. Same MODE-gate as the other engine choke points — the TS
 * branch (t20-data `pointBuySpent` + `pointBuyWarnings`) is TEST-ONLY and DCE'd
 * from the app bundle; production runs the Go/WASM `PointBuyStatusFor`. Pure
 * rules, so no catalog priming — only the loaded engine. `pointBuySpent` throws
 * on out-of-range base values (free mode), so `spent` is null there.
 */
export function pointBuyStatusFor(
  attrs: Record<AttributeKey, number>,
): PointBuyStatus {
  if (import.meta.env.MODE === 'test') {
    let spent: number | null = null
    try {
      spent = pointBuySpent(attrs)
    } catch {
      spent = null
    }
    return { spent, warnings: pointBuyWarnings(attrs) }
  }
  return enginePointBuyStatus(attrs)
}
