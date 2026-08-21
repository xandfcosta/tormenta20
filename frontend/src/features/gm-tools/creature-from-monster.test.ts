import { describe, expect, it } from 'vitest'
import type { Monster } from '@/shared/api/catalog-types'
import { creatureFromMonster } from './creature-from-monster'

const ogro = (over: Partial<Monster> = {}): Monster =>
  ({
    id: 'ogro',
    name: 'Ogro',
    nd: 4,
    tipo: 'monstro',
    size: 'grande',
    hp: 60,
    defesa: 19,
    forca: 5,
    destreza: 0,
    constituicao: 4,
    inteligencia: -2,
    sabedoria: 0,
    carisma: -1,
    fortitude: 10,
    reflexos: 3,
    vontade: 3,
    deslocamento: '9m (6q)',
    attacks: [{ name: 'Clava', attackBonus: 11, damage: '2d8+7' }],
    specialAbilities: [],
    iniciativa: 3,
    percepcao: 1,
    skills: [{ name: 'Atletismo', bonus: 12 }],
    equipamento: 'Clava',
    tesouro: 'Padrão',
    bookPage: 293,
    ...over,
  }) as Monster

/**
 * "Editar este ogro" (ALE-137): o verbete do livro é o ponto de partida do
 * bloco do mestre, porque o livro modela criatura e NPC do mesmo jeito.
 */
describe('creatureFromMonster', () => {
  it('copia os números do verbete e guarda de onde veio', () => {
    const bloco = creatureFromMonster(ogro())

    expect(bloco).toMatchObject({
      nd: 4,
      tipo: 'monstro',
      size: 'grande',
      hp: 60,
      defesa: 19,
      fortitude: 10,
      sourceMonsterId: 'ogro',
    })
    expect(bloco.attacks).toEqual([{ name: 'Clava', attackBonus: 11, damage: '2d8+7' }])
  })

  // Antes da ALE-151 o catálogo não tinha estes campos, e este arquivo os
  // deixava vazios com uma nota dizendo que voltariam. Voltaram.
  it('traz os campos do livro que a importação tinha perdido', () => {
    const bloco = creatureFromMonster(ogro())

    expect(bloco.iniciativa).toBe(3)
    expect(bloco.percepcao).toBe(1)
    expect(bloco.equipment).toBe('Clava')
    expect(bloco.treasure).toBe('Padrão')
  })

  // As perícias vinham RASPADAS do texto de `specialAbilities`, porque era ali
  // que a importação as tinha jogado. Agora são campo, e a raspagem morreu
  // junto com o motivo dela.
  it('as perícias vêm do campo, não do texto das habilidades', () => {
    const bloco = creatureFromMonster(
      ogro({
        skills: [{ name: 'Furtividade', bonus: 4, nota: '+14 em pântanos' }],
        specialAbilities: ['Faro.'],
      }),
    )

    expect(bloco.skills).toEqual([{ name: 'Furtividade', bonus: 4, nota: '+14 em pântanos' }])
    expect(bloco.specialAbilities).toEqual(['Faro.'])
  })

  // O caso do Centauro Xamã: "PM 20; Medo de Altura." misturava duas
  // informações numa frase. O PM virou campo e a frase virou o que ela é.
  it('o PM vem do campo, e só existe em conjurador', () => {
    expect(creatureFromMonster(ogro()).pm).toBeUndefined()
    expect(creatureFromMonster(ogro({ pm: 20 })).pm).toBe(20)
  })
})
