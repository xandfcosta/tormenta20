/**
 * PV/PM write-through sync — pure diff between the stored Character columns
 * and the engine-derived pools.
 *
 * Domain invariant: `hpMax`/`mpMax` are DERIVED values. The t20-data engine
 * (`computeSheetForRow`) is the single source of truth; the stored columns
 * are only a fast read cache for HUD/list views and MUST follow the engine.
 * A 2026-08 live audit found every seeded character diverged (Tanque stored
 * 82 PV vs engine 137; Necromante stored CURRENT 90 above the engine max 54)
 * because `create` persisted client-supplied maxes and no mutation ever
 * recomputed them. CharactersService now syncs after every write that
 * changes computeSheet inputs and heals stale rows on read.
 */

/** Stored PV/PM pool columns on a Character row. */
export type StoredVitals = {
  hpMax: number;
  hpCurrent: number;
  mpMax: number;
  mpCurrent: number;
};

/**
 * Diff stored PV/PM columns against the engine maxima; returns the
 * write-through patch holding ONLY the fields that changed, or `null`
 * when already in sync. Currents are clamped into [0, newMax], never
 * scaled. Unchanged fields are omitted on purpose: the heal write runs
 * outside any transaction, so echoing an untouched `hpCurrent` back
 * would clobber a concurrent vitals write with a stale read.
 *
 * @example
 * engineVitalsPatch(
 *   { hpMax: 82, hpCurrent: 82, mpMax: 43, mpCurrent: 43 },
 *   { pvMax: 137, pmMax: 30 },
 * ) // → { hpMax: 137, mpMax: 30, mpCurrent: 30 }
 */
export function engineVitalsPatch(
  stored: StoredVitals,
  engine: { pvMax: number; pmMax: number },
): Partial<StoredVitals> | null {
  const next: StoredVitals = {
    hpMax: engine.pvMax,
    hpCurrent: clampCurrent(stored.hpCurrent, engine.pvMax),
    mpMax: engine.pmMax,
    mpCurrent: clampCurrent(stored.mpCurrent, engine.pmMax),
  };
  const patch: Partial<StoredVitals> = {};
  for (const key of Object.keys(next) as (keyof StoredVitals)[]) {
    if (next[key] !== stored[key]) patch[key] = next[key];
  }
  return Object.keys(patch).length > 0 ? patch : null;
}

/** Clamp a current pool into [0, max]; valid values pass through unchanged. */
function clampCurrent(current: number, max: number): number {
  return Math.min(Math.max(0, current), max);
}
