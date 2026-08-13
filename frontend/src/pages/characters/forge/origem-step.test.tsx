import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import { createCharacterDraftStore } from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { OrigemStep } from './origem-step'

const options = {
  races: ['Humano'],
  classes: ['Guerreiro'],
  origins: ['Acólito', 'Batedor', 'Soldado'],
  gods: ['Khalmyr'],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep() {
  const draft = createCharacterDraftStore(new FakeStorage())
  render(() => (
    <ForgeProvider draft={draft} options={options}>
      <OrigemStep />
    </ForgeProvider>
  ))
  return { draft }
}

const detail = () => screen.getByRole('region', { name: /origem escolhida/i })

describe('OrigemStep', () => {
  it('lista as origens e escolhe pelo nome', async () => {
    const { draft } = renderStep()

    await userEvent.click(screen.getByRole('option', { name: 'Batedor' }))

    expect(draft.values.origin).toBe('Batedor')
  })

  it('filtra a lista pela busca', async () => {
    renderStep()

    await userEvent.type(screen.getByRole('searchbox', { name: /buscar origem/i }), 'sold')

    expect(screen.getByRole('option', { name: 'Soldado' })).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'Acólito' })).not.toBeInTheDocument()
  })

  it('convida a escolher antes de qualquer origem estar selecionada', () => {
    renderStep()

    expect(detail()).toHaveTextContent(/escolha uma origem/i)
  })

  it('marca dois benefícios e trava o terceiro', async () => {
    const { draft } = renderStep()
    await userEvent.click(screen.getByRole('option', { name: 'Soldado' }))

    const benefits = within(detail()).getAllByRole('button')
    await userEvent.click(benefits[0])
    await userEvent.click(benefits[1])

    expect(draft.values.originChoices).toHaveLength(2)
    // A cap that only refuses on click is a cap the player discovers by failing.
    expect(within(detail()).getAllByRole('button')[2]).toBeDisabled()
  })

  it('desmarcar um benefício destrava os outros de novo', async () => {
    const { draft } = renderStep()
    await userEvent.click(screen.getByRole('option', { name: 'Soldado' }))

    const benefits = () => within(detail()).getAllByRole('button')
    await userEvent.click(benefits()[0])
    await userEvent.click(benefits()[1])
    await userEvent.click(benefits()[0])

    expect(draft.values.originChoices).toHaveLength(1)
    expect(benefits()[2]).not.toBeDisabled()
  })

  it('trocar de origem limpa os benefícios da anterior', async () => {
    const { draft } = renderStep()
    await userEvent.click(screen.getByRole('option', { name: 'Soldado' }))
    await userEvent.click(within(detail()).getAllByRole('button')[0])
    expect(draft.values.originChoices).toHaveLength(1)

    await userEvent.click(screen.getByRole('option', { name: 'Acólito' }))

    expect(draft.values.origin).toBe('Acólito')
    expect(draft.values.originChoices).toEqual([])
  })

  it('diz quantos benefícios ainda faltam', async () => {
    renderStep()

    await userEvent.click(screen.getByRole('option', { name: 'Soldado' }))

    expect(detail()).toHaveTextContent(/escolha 2 benefícios/i)
  })

  it('pede o poder concreto quando o benefício é "um poder à sua escolha"', async () => {
    renderStep()
    await userEvent.click(screen.getByRole('option', { name: 'Soldado' }))

    const freePick = within(detail()).getByRole('button', { name: /poder de combate/i })
    await userEvent.click(freePick)

    expect(within(detail()).getByRole('button', { name: /escolher poder/i })).toBeInTheDocument()
  })
})
