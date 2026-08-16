import { type AttributeKey } from '@/shared/api/attribute-keys'
import {
  pointBuyStatus as enginePointBuyStatus,
  type PointBuyStatus,
} from '@/shared/lib/engine-wasm'

export type { PointBuyStatus }

/**
 * Creation point-buy status (p17) CHOKE POINT (migração TS→Go): total spent +
 * advisory warnings. Quem calcula é o Go/WASM `PointBuyStatusFor` em todos os
 * ambientes — o ramo TS do `t20-data` não existe mais (ALE-104). Regras puras,
 * sem priming de catálogo: só o motor carregado. O motor recusa base fora da
 * faixa (modo livre), e nesse caso `spent` é nulo.
 */
export function pointBuyStatusFor(
  attrs: Record<AttributeKey, number>,
): PointBuyStatus {
  return enginePointBuyStatus(attrs)
}
