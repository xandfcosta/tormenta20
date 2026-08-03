import { engineVitalsPatch } from './vitals-sync.helpers';

/**
 * Pure-diff spec for the PV/PM write-through sync (single source of truth =
 * t20-data engine). Numbers cite the 2026-08 live audit that motivated the
 * invariant: every seeded character's stored hpMax/mpMax diverged from the
 * engine (Tanque stored 82 PV vs engine 137; Necromante stored CURRENT 90
 * above the engine max 54).
 */
describe('engineVitalsPatch', () => {
  it('returns null when stored columns already match the engine', () => {
    const patch = engineVitalsPatch(
      { hpMax: 137, hpCurrent: 100, mpMax: 30, mpCurrent: 30 },
      { pvMax: 137, pmMax: 30 },
    );
    expect(patch).toBeNull();
  });

  it('rewrites stale maxes to the engine values (Tanque: 82 → 137)', () => {
    const patch = engineVitalsPatch(
      { hpMax: 82, hpCurrent: 82, mpMax: 43, mpCurrent: 43 },
      { pvMax: 137, pmMax: 30 },
    );
    // hpCurrent (82, still valid) is omitted: the heal write must never
    // echo untouched fields, or it would clobber a concurrent vitals edit.
    expect(patch).toEqual({ hpMax: 137, mpMax: 30, mpCurrent: 30 });
  });

  it('clamps currents above the new max (Necromante: current 90 > engine 54)', () => {
    const patch = engineVitalsPatch(
      { hpMax: 96, hpCurrent: 90, mpMax: 51, mpCurrent: 51 },
      { pvMax: 54, pmMax: 63 },
    );
    expect(patch).toEqual({ hpMax: 54, hpCurrent: 54, mpMax: 63 });
  });

  it('clamps negative currents to 0', () => {
    const patch = engineVitalsPatch(
      { hpMax: 21, hpCurrent: -5, mpMax: 3, mpCurrent: 3 },
      { pvMax: 21, pmMax: 3 },
    );
    expect(patch).toEqual({ hpCurrent: 0 });
  });

  it('emits a patch when only a current is out of range (max already synced)', () => {
    const patch = engineVitalsPatch(
      { hpMax: 21, hpCurrent: 30, mpMax: 3, mpCurrent: 2 },
      { pvMax: 21, pmMax: 3 },
    );
    expect(patch).toEqual({ hpCurrent: 21 });
  });
});
