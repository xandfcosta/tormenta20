import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import fixtures from './__fixtures__/character-input-parity.json'
import { assembleSheetV2, computedSheetFor } from './computed-sheet'
import { characterEffects } from './derived'

const chars = fixtures as { slug: string; char: Character }[]

/**
 * The MODE-gated `computedSheetFor` runs under vitest (MODE === 'test'), so it
 * must return the TS-assembled sheet — proving the test branch reaches the same
 * `assembleSheetV2` the parity oracle dumps, with no wasm involved.
 */
describe('computedSheetFor — test-branch parity with assembleSheetV2', () => {
  it('matches the direct assembler for every seed character', () => {
    expect(chars.length).toBeGreaterThan(0)
    for (const { char } of chars) {
      const viaHook = computedSheetFor(char)
      const direct = assembleSheetV2(char, characterEffects(char))
      expect(viaHook).toEqual(direct)
    }
  })

  it('exposes the core breakdown fields with sane shapes', () => {
    const { char } = chars[0]
    const sheet = computedSheetFor(char)
    expect(sheet.defense.total).toBeTypeOf('number')
    expect(sheet.attributes.strength.total).toBeTypeOf('number')
    expect(Array.isArray(sheet.expertises)).toBe(true)
    expect(sheet.expertises.length).toBe(char.expertises.length)
  })
})
