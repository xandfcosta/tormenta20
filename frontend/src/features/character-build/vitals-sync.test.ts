import { describe, expect, it } from 'vitest'
import { vitalsSyncPatch } from './vitals-sync'

const at = (v: { hpMax: number; hpCurrent: number; mpMax: number; mpCurrent: number }) => v

describe('vitalsSyncPatch', () => {
  it('is null when the draft already matches the derived pools', () => {
    expect(
      vitalsSyncPatch(at({ hpMax: 22, hpCurrent: 20, mpMax: 3, mpCurrent: 3 }), {
        pvMax: 22,
        pmMax: 3,
      }),
    ).toBeNull()
  })

  it('carries a full bar up with its maximum', () => {
    // Full at 22/22 and the class change makes it 27 — the player never asked
    // to start wounded, so they are full at 27, not sitting at 22/27.
    expect(
      vitalsSyncPatch(at({ hpMax: 22, hpCurrent: 22, mpMax: 3, mpCurrent: 3 }), {
        pvMax: 27,
        pmMax: 7,
      }),
    ).toEqual({ hpMax: 27, hpCurrent: 27, mpMax: 7, mpCurrent: 7 })
  })

  it('leaves a deliberately wounded current where it is', () => {
    expect(
      vitalsSyncPatch(at({ hpMax: 22, hpCurrent: 10, mpMax: 3, mpCurrent: 1 }), {
        pvMax: 27,
        pmMax: 7,
      }),
    ).toEqual({ hpMax: 27, mpMax: 7 })
  })

  it('clamps a current that the new maximum no longer holds', () => {
    expect(
      vitalsSyncPatch(at({ hpMax: 27, hpCurrent: 25, mpMax: 7, mpCurrent: 6 }), {
        pvMax: 22,
        pmMax: 3,
      }),
    ).toEqual({ hpMax: 22, hpCurrent: 22, mpMax: 3, mpCurrent: 3 })
  })

  it('touches only the pool that actually moved', () => {
    expect(
      vitalsSyncPatch(at({ hpMax: 22, hpCurrent: 22, mpMax: 3, mpCurrent: 3 }), {
        pvMax: 22,
        pmMax: 7,
      }),
    ).toEqual({ mpMax: 7, mpCurrent: 7 })
  })

  it('ignores a preview with no class yet (pvMax 0 would wipe the draft)', () => {
    expect(
      vitalsSyncPatch(at({ hpMax: 10, hpCurrent: 10, mpMax: 0, mpCurrent: 0 }), {
        pvMax: 0,
        pmMax: 0,
      }),
    ).toBeNull()
  })
})
