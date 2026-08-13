import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import type { Monster } from '@/shared/api/catalog-types'
import { describe, expect, it, vi } from 'vitest'
import { type EnrichedGroup, enrichEncounter } from './encounter'
import { EncounterComposer } from './encounter-composer'

const monster = (id: string, name: string, nd: number) => ({ id, name, nd }) as Monster
const BESTIARY = [monster('goblin', 'Goblin', 0.25), monster('ogro', 'Ogro', 2)]

const groupsOf = (entries: { monsterId: string; quantity: number }[]): EnrichedGroup[] =>
  enrichEncounter(entries, BESTIARY)

function renderComposer(groups: EnrichedGroup[], partyLevel = 1) {
  const handlers = {
    onPartyLevel: vi.fn(),
    onPartySize: vi.fn(),
    onQuantity: vi.fn(),
    onRemove: vi.fn(),
  }
  render(() => (
    <EncounterComposer
      groups={groups}
      partyLevel={partyLevel}
      partySize={4}
      {...handlers}
      addControl={<button type="button">Adicionar criatura</button>}
    />
  ))
  return handlers
}

describe('EncounterComposer', () => {
  it('convida a compor quando o encontro está vazio', () => {
    renderComposer([])

    expect(screen.getByText(/nenhuma criatura no encontro/i)).toBeInTheDocument()
  })

  it('soma o ND dos grupos e nomeia a dificuldade', () => {
    // Quatro goblins de ND 1/4 = ND 1; contra um grupo de nível 1 é justo.
    renderComposer(groupsOf([{ monsterId: 'goblin', quantity: 4 }]))

    expect(screen.getByText('1')).toBeInTheDocument()
    expect(screen.getByText('Médio')).toBeInTheDocument()
  })

  it('mostra o XP por personagem, não o total', () => {
    renderComposer(groupsOf([{ monsterId: 'ogro', quantity: 1 }]))

    // ND 2 → 2000 XP no total, dividido por 4 personagens.
    expect(screen.getByText('500')).toBeInTheDocument()
  })

  it('escreve o ND fracionário do grupo como o livro escreve', () => {
    renderComposer(groupsOf([{ monsterId: 'goblin', quantity: 2 }]))

    expect(screen.getByText(/grupo ND 1\/2/)).toBeInTheDocument()
  })

  it('ajusta a quantidade pela linha', async () => {
    const handlers = renderComposer(groupsOf([{ monsterId: 'goblin', quantity: 2 }]))

    await userEvent.click(screen.getByRole('button', { name: /aumentar goblin/i }))

    expect(handlers.onQuantity).toHaveBeenCalledWith('goblin', 3)
  })

  it('remove a criatura do encontro', async () => {
    const handlers = renderComposer(groupsOf([{ monsterId: 'ogro', quantity: 1 }]))

    await userEvent.click(screen.getByRole('button', { name: /remover ogro/i }))

    expect(handlers.onRemove).toHaveBeenCalledWith('ogro')
  })

  it('a dificuldade acompanha o nível do grupo', () => {
    renderComposer(groupsOf([{ monsterId: 'goblin', quantity: 4 }]), 6)

    // ND 1 contra nível 6 → o livro chama de irrelevante.
    expect(screen.getByText('Trivial')).toBeInTheDocument()
  })
})
