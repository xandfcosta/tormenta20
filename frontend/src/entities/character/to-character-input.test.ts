import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import fixtures from './__fixtures__/character-input-parity.json'
import { characterToInput } from './to-character-input'

/**
 * PARITY: the front `characterToInput` must produce byte-equal CharacterInput to
 * the backend `toCharacterInput`. Fixtures pair each seeded character (API shape,
 * from GET :id) with the bench payload (= backend output). Any drift between the
 * two mappers fails here — the guard that lets the front feed the SAME Go/WASM
 * engine as the server (Fase 3 → WASM).
 */
describe('characterToInput — paridade com o backend toCharacterInput', () => {
  for (const fx of fixtures as { slug: string; char: Character; expected: unknown }[]) {
    it(`${fx.slug}`, () => {
      expect(characterToInput(fx.char)).toEqual(fx.expected)
    })
  }
})
