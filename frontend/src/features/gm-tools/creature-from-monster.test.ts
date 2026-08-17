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
    treasureXp: 0,
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

  // Zero seria inventar número de livro. O editor mostra o campo vazio e o
  // mestre decide — os dados voltam pela ALE-151.
  it('deixa em branco o que a importação perdeu, em vez de chutar', () => {
    const bloco = creatureFromMonster(ogro())

    expect(bloco.equipment).toBe('')
    expect(bloco.treasure).toBe('')
    expect(bloco.iniciativa).toBe(0)
  })

  // A linha "Perícias:" É o dado, só que virou texto na importação (37 dos 80
  // verbetes). Estruturá-la evita o mestre redigitar o que o livro já diz.
  it('estrutura a linha de perícias que virou texto', () => {
    const bloco = creatureFromMonster(
      ogro({ specialAbilities: ['Perícias: Furtividade +5, Adestrar Animais +9.'] }),
    )

    expect(bloco.skills).toEqual([
      { name: 'Furtividade', bonus: 5 },
      { name: 'Adestrar Animais', bonus: 9 },
    ])
    expect(bloco.specialAbilities).toEqual([])
  })

  it('bônus negativo continua negativo', () => {
    const bloco = creatureFromMonster(ogro({ specialAbilities: ['Perícias: Furtividade -1.'] }))

    expect(bloco.skills).toEqual([{ name: 'Furtividade', bonus: -1 }])
  })

  // Perder a linha seria pior que não estruturá-la: o mestre ainda precisa
  // LER o que o livro escreveu, mesmo que o parser não entenda a forma.
  it('frase que não casa continua na lista de habilidades', () => {
    const bloco = creatureFromMonster(
      ogro({ specialAbilities: ['Perícias: veja o texto do capítulo.', 'Faro.'] }),
    )

    expect(bloco.skills).toEqual([])
    expect(bloco.specialAbilities).toEqual(['Perícias: veja o texto do capítulo.', 'Faro.'])
  })

  // O caso do Centauro Xamã: "PM 20; Medo de Altura." mistura duas informações
  // numa frase. Estruturar isso seria adivinhação — fica para a ALE-151, e a
  // frase continua visível.
  it('não tenta adivinhar PM de dentro de uma frase', () => {
    const bloco = creatureFromMonster(ogro({ specialAbilities: ['PM 20; Medo de Altura.'] }))

    expect(bloco.pm).toBeUndefined()
    expect(bloco.specialAbilities).toEqual(['PM 20; Medo de Altura.'])
  })
})
