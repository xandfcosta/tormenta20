import type { QueryClient } from '@tanstack/react-query'

/**
 * Invalidates every derived cache entry that depends on a Character row
 * — *without* touching the base `['characters', id]` key. Callers write
 * the authoritative server response into the base key via `setQueryData`
 * and then call this to refresh the derived views.
 *
 * Scope:
 *   - `['characters', id, 'sheet']`      — the ComputedSheet view; the
 *     orchestrator re-runs on the server, so a stale cache would drift
 *     from freshly-persisted state.
 *   - `['characters', id, 'campaigns']`  — the campaigns tab on the
 *     detail page (name/level render there via the include).
 *   - `['campaigns', *, 'members']`      — every campaign roster that
 *     might list this character; predicate-match saves us from listing
 *     each campaign explicitly.
 *
 * The base key is deliberately excluded: callers already `setQueryData`
 * it with the server response, and re-invalidating would trigger a
 * redundant refetch that races the just-written data.
 */
export function invalidateCharacterDependents(
  qc: QueryClient,
  characterId: number,
): void {
  qc.invalidateQueries({ queryKey: ['characters', characterId, 'sheet'] })
  qc.invalidateQueries({
    queryKey: ['characters', characterId, 'campaigns'],
  })
  qc.invalidateQueries({
    predicate: (q) =>
      q.queryKey[0] === 'campaigns' && q.queryKey[2] === 'members',
  })
}
