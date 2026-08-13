import { attributePresetForClass } from '@/shared/rules/classes'
import { ATTRIBUTE_KEYS, type AttributeKey } from '@/shared/api/attribute-keys'

export type ClassEntry = { className: string; level: number }

/** The six base attributes, apart from everything else in the draft. */
export type AttributeSpread = Record<AttributeKey, number>

const MIN_LEVEL = 1
const MAX_LEVEL = 20

/**
 * Add a class to the build. The FIRST entry is the mechanical primary — it
 * seeds PV and drives the attribute preset — so a new class always lands at
 * the end. Re-adding a class already in the list is a no-op: the wizard schema
 * rejects duplicates ("combine levels in one entry instead").
 *
 * @example addClassEntry([{ className: 'Guerreiro', level: 3 }], 'Ladino')
 */
export function addClassEntry(entries: ClassEntry[], className: string): ClassEntry[] {
  if (entries.some((entry) => entry.className === className)) return entries
  return [...entries, { className, level: MIN_LEVEL }]
}

/** Drop a class. Removing the primary promotes the next one — the build never
 *  ends up with levels but no primary class. */
export function removeClassEntry(entries: ClassEntry[], className: string): ClassEntry[] {
  return entries.filter((entry) => entry.className !== className)
}

/** Set one class's level, clamped to the book's 1..20. A NaN from an emptied
 *  number field must not reach the vitals engine, so it lands on 1. */
export function setClassLevel(
  entries: ClassEntry[],
  className: string,
  level: number,
): ClassEntry[] {
  const safe = Number.isFinite(level)
    ? Math.min(MAX_LEVEL, Math.max(MIN_LEVEL, Math.trunc(level)))
    : MIN_LEVEL
  return entries.map((entry) =>
    entry.className === className ? { ...entry, level: safe } : entry,
  )
}

/** Character level = the levels summed across classes (p34-35). A build with
 *  no class yet is still a level 1 character, which is what the previews use. */
export function totalClassLevel(entries: ClassEntry[]): number {
  return entries.reduce((sum, entry) => sum + (entry.level || 0), 0) || MIN_LEVEL
}

/** The suggested attribute spread for a class, or null for an unknown one. */
export function classPresetSpread(className: string): AttributeSpread | null {
  return attributePresetForClass(className)
}

/**
 * Snapshot just the six attributes out of the wider draft — what an "undo" has
 * to restore, without dragging back the name or the races chosen since. Takes
 * the whole draft (that is the caller), so a non-numeric field lands on 0
 * rather than poisoning the spread.
 *
 * @example const previous = attributeSpreadOf(draft.values)
 */
export function attributeSpreadOf(values: Readonly<Record<string, unknown>>): AttributeSpread {
  const out = {} as AttributeSpread
  for (const key of ATTRIBUTE_KEYS) {
    const value = values[key]
    out[key] = typeof value === 'number' ? value : 0
  }
  return out
}
