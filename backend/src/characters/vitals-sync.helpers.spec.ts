import { engineVitalsPatch, levelVitalsPatch } from './vitals-sync.helpers';

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

/**
 * Level-change variant: currents FOLLOW the max delta instead of only being
 * clamped — leveling up must not leave the character "wounded" by the new,
 * larger max (owner report 2026-08: 68/87 → level up → 68/96 instead of
 * 77/96). Leveling down walks the same delta back, floored at 0.
 */
describe('levelVitalsPatch', () => {
  it('shifts currents up by the max delta on level up (68/87 → 77/96)', () => {
    const patch = levelVitalsPatch(
      { hpMax: 87, hpCurrent: 68, mpMax: 24, mpCurrent: 24 },
      { pvMax: 96, pmMax: 27 },
    );
    expect(patch).toEqual({
      hpMax: 96,
      hpCurrent: 77,
      mpMax: 27,
      mpCurrent: 27,
    });
  });

  it('shifts currents down by the max delta on level down (77/96 → 68/87)', () => {
    const patch = levelVitalsPatch(
      { hpMax: 96, hpCurrent: 77, mpMax: 27, mpCurrent: 27 },
      { pvMax: 87, pmMax: 24 },
    );
    expect(patch).toEqual({
      hpMax: 87,
      hpCurrent: 68,
      mpMax: 24,
      mpCurrent: 24,
    });
  });

  it('floors a shifted current at 0 on level down', () => {
    const patch = levelVitalsPatch(
      { hpMax: 96, hpCurrent: 4, mpMax: 27, mpCurrent: 1 },
      { pvMax: 87, pmMax: 24 },
    );
    expect(patch).toEqual({ hpMax: 87, hpCurrent: 0, mpMax: 24, mpCurrent: 0 });
  });

  it('caps a shifted current at the new max', () => {
    // Stored current above the old max (e.g. temp effects) still lands ≤ max.
    const patch = levelVitalsPatch(
      { hpMax: 87, hpCurrent: 90, mpMax: 24, mpCurrent: 24 },
      { pvMax: 96, pmMax: 27 },
    );
    expect(patch).toEqual({
      hpMax: 96,
      hpCurrent: 96,
      mpMax: 27,
      mpCurrent: 27,
    });
  });

  it('returns null when the engine maxes match the stored ones', () => {
    const patch = levelVitalsPatch(
      { hpMax: 87, hpCurrent: 68, mpMax: 24, mpCurrent: 24 },
      { pvMax: 87, pmMax: 24 },
    );
    expect(patch).toBeNull();
  });
});
