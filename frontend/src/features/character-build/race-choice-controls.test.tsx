import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { RaceChoice } from './grant-helpers'
import { RaceChoiceControls } from './race-choice-controls'

function renderControls(raceName: string, choice: RaceChoice = {}) {
  const onChange = vi.fn()
  render(() => (
    <RaceChoiceControls raceName={raceName} choice={choice} onChange={onChange} />
  ))
  return { onChange }
}

describe('RaceChoiceControls — raça de bônus fixo', () => {
  it('não pede escolha nenhuma (Anão é +2 CON e pronto)', () => {
    renderControls('Anão')

    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})

describe('RaceChoiceControls — +1 flutuante (Humano)', () => {
  it('conta quantos já foram colocados', () => {
    renderControls('Humano', { floatingPicks: ['strength'] })

    expect(screen.getByText(/1\/3/)).toBeInTheDocument()
  })

  it('colocar um +1 acrescenta o atributo', async () => {
    const { onChange } = renderControls('Humano')

    await userEvent.click(screen.getByRole('button', { name: 'FOR' }))

    expect(onChange).toHaveBeenCalledWith({ floatingPicks: ['strength'] })
  })

  it('clicar de novo tira o +1 (é alternância, não acúmulo)', async () => {
    const { onChange } = renderControls('Humano', { floatingPicks: ['strength'] })

    await userEvent.click(screen.getByRole('button', { name: 'FOR' }))

    expect(onChange).toHaveBeenCalledWith({ floatingPicks: [] })
  })

  it('com a cota cheia, os atributos livres ficam indisponíveis', async () => {
    const { onChange } = renderControls('Humano', {
      floatingPicks: ['strength', 'dexterity', 'constitution'],
    })

    const untaken = screen.getByRole('button', { name: 'INT' })
    expect(untaken).toBeDisabled()
    await userEvent.click(untaken)
    expect(onChange).not.toHaveBeenCalled()

    // Mas o que já foi colocado continua clicável — senão não dá pra desfazer.
    expect(screen.getByRole('button', { name: 'FOR' })).toBeEnabled()
  })

  it('preserva o resto da escolha ao mexer nos atributos', async () => {
    const { onChange } = renderControls('Humano', { applied: true })

    await userEvent.click(screen.getByRole('button', { name: 'CAR' }))

    expect(onChange).toHaveBeenCalledWith({ applied: true, floatingPicks: ['charisma'] })
  })
})

describe('RaceChoiceControls — ascendência (Suraggel)', () => {
  it('oferece as ascendências com o que cada uma vale', () => {
    renderControls('Suraggel')

    expect(screen.getByRole('button', { name: /aggelus/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sulfure/i })).toBeInTheDocument()
  })

  it('escolher uma ascendência grava a escolha', async () => {
    const { onChange } = renderControls('Suraggel')

    await userEvent.click(screen.getByRole('button', { name: /aggelus/i }))

    expect(onChange).toHaveBeenCalledWith({ ascendencia: 'aggelus' })
  })

  it('marca a escolhida como pressionada', () => {
    renderControls('Suraggel', { ascendencia: 'aggelus' })

    expect(screen.getByRole('button', { name: /aggelus/i })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })
})
