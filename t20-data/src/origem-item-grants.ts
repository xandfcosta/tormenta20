import { ORIGENS } from './origens'
// The pure parser + OrigemItemGrant type live in ./origem-item-parse (data-free)
// so the front can call them off its cached origens without anchoring ORIGENS.
export { parseOrigemItem } from './origem-item-parse'
export type { OrigemItemGrant } from './origem-item-parse'
import { type OrigemItemGrant, parseOrigemItem } from './origem-item-parse'

/** Structured item grants for an origem, by NAME (as the app stores it). */
export function origemItemGrantsByName(
  originName: string,
): OrigemItemGrant[] {
  const origem = Object.values(ORIGENS).find((o) => o.name === originName)
  return (origem?.itensIniciais ?? []).map(parseOrigemItem)
}
