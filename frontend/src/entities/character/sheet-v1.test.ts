import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Character } from '@/shared/api/api'

/** Named fake for the WASM engine — counts computes so the memo is testable. */
class FakeEngine {
  calls: unknown[] = []
  computeSheet = (input: unknown) => {
    this.calls.push(input)
    return { level: 7, className: 'Guerreiro', warnings: [] }
  }
}

const engine = new FakeEngine()

vi.mock('@/shared/lib/engine-wasm', () => ({
  computeSheet: (input: unknown) => engine.computeSheet(input),
}))
vi.mock('./to-character-input', () => ({
  characterToInput: (character: Character) => ({ level: character.level }),
}))

const { computedSheetV1For } = await import('./sheet-v1')

/** Only the fields this unit touches — the engine call itself is faked. */
function character(overrides: Partial<Character> = {}): Character {
  return { id: 1, name: 'Tanque', level: 7, classes: [], ...overrides } as Character
}

beforeEach(() => {
  engine.calls = []
})

describe('computedSheetV1For', () => {
  // Regressão ALE-77: essas telas liam `data.computed` de um endpoint que o
  // backend Go não devolve mais nesse shape. Agora derivam pelo MESMO motor Go,
  // só que via WASM — sem ida ao servidor e sem shape para desalinhar.
  it('deriva a ficha pelo motor a partir do personagem', () => {
    const sheet = computedSheetV1For(character())
    expect(sheet.level).toBe(7)
    expect(engine.calls).toHaveLength(1)
  })

  // Várias cartas da ficha pedem a mesma derivação no mesmo render; sem memo
  // cada uma reexecutaria o WASM.
  it('memoiza por personagem', () => {
    const hero = character()
    computedSheetV1For(hero)
    computedSheetV1For(hero)
    expect(engine.calls).toHaveLength(1)
  })

  it('recomputa quando o personagem é outro objeto (edição → query nova)', () => {
    computedSheetV1For(character())
    computedSheetV1For(character({ level: 8 }))
    expect(engine.calls).toHaveLength(2)
  })
})
