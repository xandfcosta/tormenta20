import { createSignal } from 'solid-js'
import type { EncounterEntry } from './encounter'

export type EncounterDraft = {
  entries: () => EncounterEntry[]
  partyLevel: () => number
  partySize: () => number
  setPartyLevel: (level: number) => void
  setPartySize: (size: number) => void
  /** Adds one of a creature, or bumps the count if it is already in. */
  add: (monsterId: string) => void
  setQuantity: (monsterId: string, quantity: number) => void
  remove: (monsterId: string) => void
  clear: () => void
}

const DEFAULT_PARTY_LEVEL = 1
const DEFAULT_PARTY_SIZE = 4

/**
 * The encounter being composed. `create*` holding state → born once in a
 * component body (gotcha #17).
 *
 * Adding a creature already in the encounter BUMPS its count instead of adding
 * a second row: two rows of the same monster would each compute their own group
 * ND, and the p282 doubling rule only means anything over one group.
 *
 * @example const draft = createEncounterDraft(); draft.add('goblin')
 */
export function createEncounterDraft(): EncounterDraft {
  const [entries, setEntries] = createSignal<EncounterEntry[]>([])
  const [partyLevel, setPartyLevel] = createSignal(DEFAULT_PARTY_LEVEL)
  const [partySize, setPartySize] = createSignal(DEFAULT_PARTY_SIZE)

  return {
    entries,
    partyLevel,
    partySize,
    setPartyLevel: (level) => setPartyLevel(clamp(level, 1, 20)),
    setPartySize: (size) => setPartySize(clamp(size, 1, 8)),
    add: (monsterId) =>
      setEntries((prev) =>
        prev.some((entry) => entry.monsterId === monsterId)
          ? prev.map((entry) =>
              entry.monsterId === monsterId
                ? { ...entry, quantity: entry.quantity + 1 }
                : entry,
            )
          : [...prev, { monsterId, quantity: 1 }],
      ),
    setQuantity: (monsterId, quantity) =>
      setEntries((prev) =>
        prev.map((entry) =>
          entry.monsterId === monsterId
            ? { ...entry, quantity: Math.max(1, quantity) }
            : entry,
        ),
      ),
    remove: (monsterId) =>
      setEntries((prev) => prev.filter((entry) => entry.monsterId !== monsterId)),
    clear: () => setEntries([]),
  }
}

function clamp(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min
  return Math.min(max, Math.max(min, Math.round(value)))
}
