import { describe, expect, it } from 'vitest'
import { healthVignette } from './health-vignette'

describe('healthVignette', () => {
  it('is none while at or above half HP', () => {
    expect(healthVignette(40, 40)).toEqual({ kind: 'none' })
    expect(healthVignette(20, 40)).toEqual({ kind: 'none' })
  })

  it('ramps the wound from just under half toward zero', () => {
    // 10/40 = 25% → halfway down the (50%→0%) ramp
    expect(healthVignette(10, 40)).toEqual({ kind: 'wound', t: 0.5 })
    // 4/40 = 10% → most of the way
    expect(healthVignette(4, 40)).toEqual({ kind: 'wound', t: 0.8 })
  })

  it('is dying at 0 or below (agonizing)', () => {
    expect(healthVignette(0, 40)).toEqual({ kind: 'dying' })
    expect(healthVignette(-7, 40)).toEqual({ kind: 'dying' })
  })
})
