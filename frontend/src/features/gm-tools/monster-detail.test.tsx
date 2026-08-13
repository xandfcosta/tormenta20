import { render, screen } from '@solidjs/testing-library'
import type { Monster } from '@/shared/api/catalog-types'
import { describe, expect, it } from 'vitest'
import { MonsterDetail } from './monster-detail'

const GOBLIN = {
  id: 'goblin-salteador',
  name: 'Goblin Salteador',
  nd: 0.25,
  tipo: 'humanoide',
  size: 'pequeno',
  hp: 4,
  defesa: 13,
  forca: 0,
  destreza: 3,
  constituicao: 0,
  inteligencia: 0,
  sabedoria: -1,
  carisma: -1,
  fortitude: 2,
  reflexos: 3,
  vontade: -1,
  deslocamento: '9m, escalada 9m',
  attacks: [{ name: 'Duas adagas', attackBonus: 7, damage: '1d4', special: 'crítico 19' }],
  specialAbilities: ['Visão no escuro.'],
  bookPage: 300,
} as unknown as Monster

describe('MonsterDetail', () => {
  it('escreve o ND fracionário como o livro escreve', () => {
    render(() => <MonsterDetail monster={GOBLIN} />)

    expect(screen.getByText(/ND 1\/4/)).toBeInTheDocument()
  })

  it('mostra o XP de tesouro derivado do ND', () => {
    render(() => <MonsterDetail monster={GOBLIN} />)

    expect(screen.getByText(/XP 250/)).toBeInTheDocument()
  })

  // Um "0" cru ao lado de um "3" cru não diz que são modificadores.
  it('assina os modificadores, positivos e negativos', () => {
    render(() => <MonsterDetail monster={GOBLIN} />)

    // Des +3 e Reflexos +3 — os dois assinados.
    expect(screen.getAllByText('+3')).toHaveLength(2)
    expect(screen.getAllByText('-1').length).toBeGreaterThan(0)
    expect(screen.getAllByText('+0').length).toBeGreaterThan(0)
  })

  it('traz o ataque com bônus, dano e a nota especial', () => {
    render(() => <MonsterDetail monster={GOBLIN} />)

    expect(screen.getByText('Duas adagas')).toBeInTheDocument()
    expect(screen.getByText(/\+7 · 1d4/)).toBeInTheDocument()
    expect(screen.getByText('crítico 19')).toBeInTheDocument()
  })

  it('omite as seções vazias em vez de mostrar título sozinho', () => {
    const mudo = { ...GOBLIN, attacks: [], specialAbilities: [] } as unknown as Monster
    render(() => <MonsterDetail monster={mudo} />)

    expect(screen.queryByText('Ataques')).not.toBeInTheDocument()
    expect(screen.queryByText('Habilidades especiais')).not.toBeInTheDocument()
  })

  it('leva a página do livro — o mestre vai conferir na mesa', () => {
    render(() => <MonsterDetail monster={GOBLIN} />)

    expect(screen.getByText(/p300/)).toBeInTheDocument()
  })
})
