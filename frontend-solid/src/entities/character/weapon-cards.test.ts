import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import fixtures from './__fixtures__/character-input-parity.json'
import { assembleWeaponCards } from './weapon-cards'
import { attributeTotal, characterEffects } from './derived'

const base = (fixtures as { char: Character }[])[0]!.char

/** A one-weapon character wielding `catalogId`, with the given attrs + power. */
function card(
  catalogId: string,
  opts: { str?: number; dex?: number; acuidade?: boolean } = {},
) {
  const { str = 1, dex = 3, acuidade = false } = opts
  const char: Character = {
    ...base,
    strength: str,
    dexterity: dex,
    level: 2,
    classPowers: acuidade ? JSON.stringify(['acuidade-com-arma']) : '[]',
    items: [
      {
        id: 1,
        catalogId,
        name: catalogId,
        quantity: 1,
        slots: 1,
        equipped: 'wielded',
        improvements: '[]',
        material: null,
      } as Character['items'][number],
    ],
  }
  const effects = characterEffects(char)
  const c = assembleWeaponCards(char, effects)[0]!
  return {
    ...c,
    forTotal: attributeTotal(char, 'strength', effects),
    dexTotal: attributeTotal(char, 'dexterity', effects),
  }
}

// ALE-31 — Destreza no ataque (Adaga inerente / Acuidade com Arma).
describe('assembleWeaponCards — finesse (ALE-31)', () => {
  it('Adaga usa Destreza no ataque quando DES > FOR (finesse inerente)', () => {
    const c = card('adaga', { str: 1, dex: 3 })
    expect(c.skill).toBe('Luta')
    expect(c.attribute).toBe('dexterity')
  })

  it('Adaga usa Força quando FOR >= DES (o melhor atributo)', () => {
    const c = card('adaga', { str: 3, dex: 1 })
    expect(c.attribute).toBe('strength')
  })

  it('Adaga (sem Acuidade): dano continua com Força; com Acuidade vira Destreza', () => {
    const semAcu = card('adaga', { str: 1, dex: 3 })
    const comAcu = card('adaga', { str: 1, dex: 3, acuidade: true })
    expect(semAcu.strDamage).toBe(semAcu.forTotal) // dano = Força (finesse inerente é só ataque)
    expect(comAcu.strDamage).toBe(comAcu.dexTotal) // Acuidade leva Destreza pro dano
    expect(comAcu.strDamage).toBeGreaterThan(semAcu.strDamage)
  })

  it('Espada curta (leve, sem finesse) usa Força — a menos que tenha Acuidade', () => {
    expect(card('espada-curta', { str: 1, dex: 3 }).attribute).toBe('strength')
    expect(
      card('espada-curta', { str: 1, dex: 3, acuidade: true }).attribute,
    ).toBe('dexterity')
  })

  it('arma à distância continua em Destreza (Pontaria), sem depender de finesse', () => {
    expect(card('pistola', { str: 3, dex: 1 }).skill).toBe('Pontaria')
    expect(card('pistola', { str: 3, dex: 1 }).attribute).toBe('dexterity')
  })
})
