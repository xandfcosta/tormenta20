import { render, screen, within } from '@solidjs/testing-library'
import { describe, expect, it } from 'vitest'
import type { Monster } from '@/shared/api/catalog-types'
import { MonsterStatBlock } from './monster-stat-block'

/**
 * O bloco do monstro na cena do mestre (ALE-122).
 *
 * Abrir um NPC mostrava só a barra de PV — para saber a Defesa, o ataque ou a
 * habilidade especial, o mestre saía da tela e ia procurar no bestiário, no meio
 * do combate. A ordem aqui é a que ele pediu: PV (que muda a cada golpe) e
 * ataques (o que ele usa no turno) primeiro; ND, atributos e habilidades depois.
 */
const OGRO = {
  id: 'ogro',
  name: 'Ogro',
  nd: 2,
  tipo: 'monstro',
  size: 'grande',
  hp: 130,
  defesa: 15,
  forca: 5,
  destreza: 0,
  constituicao: 3,
  inteligencia: -2,
  sabedoria: 0,
  carisma: -2,
  fortitude: 7,
  reflexos: 2,
  vontade: 2,
  deslocamento: '9m',
  attacks: [
    { name: 'Clava', attackBonus: 9, damage: '2d8+7' },
    { name: 'Pedrada', attackBonus: 4, damage: '1d8+5', special: 'arremesso 9m' },
  ],
  specialAbilities: ['Faro.', 'Vulnerabilidade a luz: -2 em ataques sob luz do sol.'],
  iniciativa: 4,
  percepcao: 1,
  skills: [
    { name: 'Atletismo', bonus: 12 },
    { name: 'Furtividade', bonus: 4, nota: '+14 em pântanos' },
  ],
  equipamento: 'Clava, peles',
  tesouro: 'Padrão',
  bookPage: 300,
} as unknown as Monster

describe('MonsterStatBlock', () => {
  it('põe os ATAQUES antes da ficha, com bônus e dano', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const ataques = within(screen.getByRole('region', { name: 'Ataques' }))
    expect(ataques.getByText('Clava')).toBeInTheDocument()
    expect(ataques.getByText('+9')).toBeInTheDocument()
    expect(ataques.getByText('2d8+7')).toBeInTheDocument()
    // O especial vai junto: "arremesso 9m" muda o que o mestre pode fazer.
    expect(ataques.getByText(/arremesso 9m/)).toBeInTheDocument()
  })

  it('mostra a linha de identidade com ND, Defesa e deslocamento', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    expect(screen.getByText(/ND 2/)).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
    expect(screen.getByText('9m')).toBeInTheDocument()
    // A página do livro fica à mão para quem quer ler a entrada inteira.
    expect(screen.getByText(/p300/)).toBeInTheDocument()
  })

  // Modificador negativo tem de aparecer COM o sinal: "-2" e "2" são coisas
  // diferentes na hora de somar um teste.
  it('assina os modificadores, inclusive os negativos', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const atributos = within(screen.getByRole('region', { name: 'Atributos e resistências' }))
    expect(atributos.getByText('+5')).toBeInTheDocument() // For
    expect(atributos.getByText('+7')).toBeInTheDocument() // Fortitude
    // INT e CAR são -2 os dois: o que importa é que o SINAL apareça nos dois.
    expect(atributos.getAllByText('-2')).toHaveLength(2)
    // DES e SAB são 0: zero aparece como "+0", não como vazio nem "0".
    expect(atributos.getAllByText('+0')).toHaveLength(2)
  })

  it('lista as habilidades especiais', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const habilidades = within(screen.getByRole('region', { name: 'Habilidades' }))
    expect(habilidades.getByText(/Faro/)).toBeInTheDocument()
    expect(habilidades.getByText(/Vulnerabilidade a luz/)).toBeInTheDocument()
  })

  // Um monstro sem habilidade especial não pode deixar um título órfão na tela.
  it('sem habilidades, não rende a seção vazia', () => {
    render(() => <MonsterStatBlock monster={{ ...OGRO, specialAbilities: [] }} />)

    expect(screen.queryByRole('region', { name: 'Habilidades' })).not.toBeInTheDocument()
  })
})

/**
 * As linhas do bloco impresso que a importação tinha perdido (ALE-151).
 *
 * Iniciativa e Percepção abrem o bloco no livro e são as duas primeiras coisas
 * que o mestre rola; Equipamento se perdeu INTEIRO — zero dos 80 verbetes o
 * tinham — e Tesouro virava um número de XP que não era tesouro nenhum.
 */
describe('MonsterStatBlock — o que voltou do livro', () => {
  it('abre com Iniciativa e Percepção, como o bloco impresso', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const identidade = within(screen.getByRole('region', { name: 'Identidade' }))
    expect(identidade.getByText(/INI/)).toBeInTheDocument()
    expect(identidade.getByText('+4')).toBeInTheDocument()
    expect(identidade.getByText(/PER/)).toBeInTheDocument()
  })

  it('mostra as perícias com o bônus condicional entre parênteses', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const pericias = within(screen.getByRole('region', { name: 'Perícias' }))
    expect(pericias.getByText(/Atletismo/)).toBeInTheDocument()
    expect(pericias.getByText(/\+14 em pântanos/)).toBeInTheDocument()
  })

  it('fecha com Equipamento e Tesouro', () => {
    render(() => <MonsterStatBlock monster={OGRO} />)

    const fim = within(screen.getByRole('region', { name: 'Equipamento e tesouro' }))
    expect(fim.getByText(/Clava, peles/)).toBeInTheDocument()
    expect(fim.getByText(/Padrão/)).toBeInTheDocument()
  })
})
