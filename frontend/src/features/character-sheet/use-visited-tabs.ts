import { useRef } from 'react'

/**
 * Tracks which sheet blocks have been opened at least once so a layout can
 * `forceMount` them and keep them alive: a revisit then toggles visibility
 * instead of re-mounting the heavy panel. Re-mounting Perícias' ~850 nodes on
 * every tab switch was the tab-switch jank (see ALE-48 perf notes).
 *
 * Backed by a ref (not state) and grown DURING render: the currently active
 * block is always included, and the mutation is idempotent, so it needs no
 * extra render — an effect+setState here added a second layout render per
 * switch, which measurably widened the switch's blocking task.
 *
 * @example
 * const visited = useVisitedTabs(active)
 * <TabsContent forceMount={visited.has(s.value) || undefined} />
 */
export function useVisitedTabs(active: string): ReadonlySet<string> {
  const visited = useRef<Set<string>>(new Set())
  visited.current.add(active)
  return visited.current
}
