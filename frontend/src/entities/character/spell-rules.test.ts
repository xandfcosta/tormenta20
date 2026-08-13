import type { AttributeKey, SpellcasterClass } from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import { makeCharacter } from './__fixtures__/character'
import { bestSpellCd, castableClassesFor, highestCastableCircle, spellPmLimitFor } from './spell-rules'

const CD: Record<AttributeKey, number> = {
  strength: 10,
  dexterity: 11,
  constitution: 12,
  intelligence: 18,
  wisdom: 15,
  charisma: 14,
}

describe('castableClassesFor', () => {
  it('cruza as classes conjuradoras do personagem com a lista da magia', () => {
    const character = makeCharacter({
      classes: [
        { className: 'Arcanista', level: 5 },
        { className: 'Guerreiro', level: 2 },
      ],
    })

    expect(castableClassesFor(character, ['Arcanista', 'Clérigo'])).toEqual(['Arcanista'])
  })

  it('classe não conjuradora nunca entra', () => {
    const character = makeCharacter({ classes: [{ className: 'Guerreiro', level: 5 }] })

    expect(castableClassesFor(character, ['Arcanista'])).toEqual([])
  })
})

describe('bestSpellCd', () => {
  // Multiclasse conjurador: vale a MELHOR CD entre as classes que podem lançar
  // a magia — cada uma lança pelo seu próprio atributo-chave (p171).
  it('escolhe a maior CD entre as classes aplicáveis', () => {
    const classes: SpellcasterClass[] = ['Arcanista', 'Bardo']

    // Arcanista lança por Int (18), Bardo por Car (14).
    expect(bestSpellCd(classes, CD)).toBe(18)
  })

  it('sem classe aplicável não há CD', () => {
    expect(bestSpellCd([], CD)).toBeNull()
  })
})

describe('highestCastableCircle', () => {
  it('cresce com o nível da classe conjuradora', () => {
    const nivel1 = makeCharacter({ classes: [{ className: 'Arcanista', level: 1 }] })
    const nivel9 = makeCharacter({ classes: [{ className: 'Arcanista', level: 9 }] })

    expect(highestCastableCircle(nivel9, ['Arcanista'])).toBeGreaterThan(
      highestCastableCircle(nivel1, ['Arcanista']),
    )
  })

  // Multiclasse: o círculo vem do nível NA classe conjuradora, não do total.
  it('conta o nível na classe, não o nível do personagem', () => {
    const character = makeCharacter({
      classes: [
        { className: 'Arcanista', level: 1 },
        { className: 'Guerreiro', level: 9 },
      ],
    })

    expect(highestCastableCircle(character, ['Arcanista'])).toBe(
      highestCastableCircle(
        makeCharacter({ classes: [{ className: 'Arcanista', level: 1 }] }),
        ['Arcanista'],
      ),
    )
  })

  it('sem classe aplicável, nada além de truque', () => {
    const character = makeCharacter({ classes: [{ className: 'Guerreiro', level: 20 }] })

    expect(highestCastableCircle(character, [])).toBe(0)
  })
})

/**
 * ALE-92. O teto por magia é o número que o portão de conjurar usa, e ele
 * DIVERGIA do servidor: a ficha oferecia o resumo do HUD ("maior nível
 * conjurador") enquanto o servidor cobrava o nível na classe que fornece a
 * magia. Multiclasse é o único caso que revela a diferença.
 */
describe('spellPmLimitFor (p224)', () => {
  it('usa o nível NA CLASSE que fornece a magia, cheio', () => {
    const arcanista11 = makeCharacter({ classes: [{ className: 'Arcanista', level: 11 }], level: 11 })

    expect(spellPmLimitFor(arcanista11, ['Arcanista'])).toBe(11)
  })

  // O caso da issue: com o resumo do HUD isto daria 7, e o servidor recusaria
  // qualquer aprimoramento acima de 1.
  it('multiclasse conta só a classe da magia, não a maior', () => {
    const bardoArcanista = makeCharacter({
      classes: [
        { className: 'Bardo', level: 7 },
        { className: 'Arcanista', level: 1 },
      ],
      level: 8,
    })

    expect(spellPmLimitFor(bardoArcanista, ['Arcanista'])).toBe(1)
    expect(spellPmLimitFor(bardoArcanista, ['Bardo'])).toBe(7)
  })

  it('magia de fonte que não é classe usa o nível de personagem', () => {
    const barbaro8 = makeCharacter({ classes: [{ className: 'Bárbaro', level: 8 }], level: 8 })

    expect(spellPmLimitFor(barbaro8, ['Arcanista'])).toBe(8)
  })

  it('nunca abaixo de 1', () => {
    expect(spellPmLimitFor(makeCharacter({ classes: [], level: 0 }), [])).toBe(1)
  })
})
