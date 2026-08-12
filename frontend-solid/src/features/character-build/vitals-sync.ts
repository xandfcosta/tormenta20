import type { DraftVitals } from './draft-vitals'

type VitalFields = {
  hpMax: number
  hpCurrent: number
  mpMax: number
  mpCurrent: number
}

export type VitalsSyncPatch = Partial<VitalFields>

/**
 * Keeps the draft's PV/PM in step with the pools the build derives. Máximos are
 * never typed in — they come from classe + Constituição + nível + poderes — so
 * this is what writes them, and the saved character matches the sheet.
 *
 * The editable "atual" follows only when it was FULL (the player never asked to
 * start wounded) or when the new maximum can no longer hold it. A deliberately
 * wounded current is left alone.
 *
 * Returns null when nothing needs to move, so a caller can write only on change
 * instead of on every pass.
 *
 * @example vitalsSyncPatch(draft.values, deriveDraftVitals(...)) // { hpMax: 27, hpCurrent: 27 }
 */
export function vitalsSyncPatch(
  values: VitalFields,
  derived: DraftVitals,
): VitalsSyncPatch | null {
  // No primary class yet → the preview reports 0/0, which is absence of an
  // answer, not an answer. Writing it would wipe the draft's starting pools.
  if (derived.pvMax <= 0) return null

  const patch: VitalsSyncPatch = {
    ...poolPatch('hpMax', 'hpCurrent', values.hpMax, values.hpCurrent, derived.pvMax),
    ...poolPatch('mpMax', 'mpCurrent', values.mpMax, values.mpCurrent, derived.pmMax),
  }
  return Object.keys(patch).length > 0 ? patch : null
}

/** One pool's move: the new max, plus the current only when it has to follow. */
function poolPatch(
  maxKey: 'hpMax' | 'mpMax',
  currentKey: 'hpCurrent' | 'mpCurrent',
  max: number,
  current: number,
  nextMax: number,
): VitalsSyncPatch {
  if (max === nextMax) return {}
  const wasFull = current >= max
  return wasFull || current > nextMax
    ? { [maxKey]: nextMax, [currentKey]: nextMax }
    : { [maxKey]: nextMax }
}
