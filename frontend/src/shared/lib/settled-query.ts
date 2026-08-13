/**
 * Reads a solid-query result WITHOUT suspending: `null` while the query is
 * pending, the data once it has settled.
 *
 * Why this exists (ALE-95). `@tanstack/solid-query` backs `.data` with a
 * resource, so touching it while the query is pending SUSPENDS. The app
 * declares no `Suspense` of its own, so the nearest boundary is the one
 * `@tanstack/solid-router` puts around every route match — and when it
 * suspends, Solid detaches the whole match subtree and re-inserts it on
 * resolve. Re-inserting a node RESTARTS every CSS animation under it, so the
 * entire scene replayed its enter animation (`scene-in` fade + every
 * `animate-in`) on each selection change. The DOM nodes are identical before
 * and after, which is why this reads as "the screen flashed" and not as a
 * remount: only `getAnimations()[].startTime` moves.
 *
 * It bites whenever a query KEY changes from inside a scene, with no
 * navigation — the roster's per-character computed sheet being the loud case
 * (one flash per arrow key).
 *
 * Structural parameter on purpose: the thin seam this project owns over
 * solid-query, and it keeps the helper testable with a plain object.
 *
 * @example
 * const sheet = useQuery(() => ({ ...characterSheetQueryOptions(id()) }))
 * const computed = () => settledQuery(sheet) // null while in flight
 */
export function settledQuery<T>(query: { isPending: boolean; data: T | undefined }): T | null {
  if (query.isPending) return null
  return query.data ?? null
}
