import { createSignal } from 'solid-js'

/** How many past draws a table keeps. Enough to compare, short enough to read. */
const HISTORY_DEPTH = 5

export type RollRecord<T> = { roll: number; result: T }

export type RollHistory<T> = {
  entries: () => RollRecord<T>[]
  latest: () => RollRecord<T> | undefined
  push: (roll: number, result: T) => void
  clear: () => void
}

/**
 * A table's recent draws, newest first. Kept because a GM rolling on the same
 * table twice in a scene wants to see what came before — a single "last result"
 * throws away the comparison the moment they roll again.
 *
 * `create*` holding state → born once in a component body (gotcha #17).
 *
 * @example const ruina = createRollHistory<RuinaRow>(); ruina.push(4, row)
 */
export function createRollHistory<T>(): RollHistory<T> {
  const [entries, setEntries] = createSignal<RollRecord<T>[]>([])

  return {
    entries,
    latest: () => entries()[0],
    push: (roll, result) =>
      setEntries((prev) => [{ roll, result }, ...prev].slice(0, HISTORY_DEPTH)),
    clear: () => setEntries([]),
  }
}
