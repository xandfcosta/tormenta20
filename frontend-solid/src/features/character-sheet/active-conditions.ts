import type { ConditionId } from '@tormenta20/t20-data'
import { conditionsRecord } from '@/shared/lib/rules-catalog-cache'

/**
 * Parse the persisted `ConditionId[]` blob (bad blob ⇒ none).
 *
 * Lived inside the React `conditions-section.tsx` panel, which meant anything
 * wanting to read a character's conditions — the search index, for one — had to
 * import a component that drags React Query and the UI kit along. It is a
 * parser; it lives on its own.
 *
 * Unknown ids are dropped: the catalog is the authority on what a condition is,
 * and a stale blob must not inject a phantom one into the sheet.
 *
 * @example parseActiveConditions('["cego","lixo"]') // ['cego']
 */
export function parseActiveConditions(raw: string): ConditionId[] {
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is ConditionId => typeof x === 'string' && x in conditionsRecord())
  } catch {
    return []
  }
}
