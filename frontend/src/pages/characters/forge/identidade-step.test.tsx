import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { IdentidadeStep } from './identidade-step'

const OPTIONS = {
  races: ['Humano'],
  classes: ['Guerreiro'],
  origins: ['Acólito'],
  gods: ['Khalmyr', 'Marah'],
  sizes: ['Pequeno', 'Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  draft.setValue('classes', [{ className: 'Guerreiro', level: 2 }])
  draft.setValue('races', ['Humano'])
  draft.setValue('origin', 'Acólito')
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <IdentidadeStep />
    </ForgeProvider>
  ))
  return { draft }
}

const nameField = () => screen.getByRole('textbox', { name: /nome/i })

describe('IdentidadeStep', () => {
  it('batiza o personagem', async () => {
    const { draft } = renderStep()

    await userEvent.type(nameField(), 'Thal')

    expect(draft.values.name).toBe('Thal')
  })

  it('não perde o foco do nome a cada tecla', async () => {
    renderStep()

    await userEvent.type(nameField(), 'Thal, o Errante')

    // The frontispiece re-renders the lineage line under the field on every
    // keystroke; the field itself must survive it (gotcha #27).
    expect(nameField()).toHaveFocus()
    expect(nameField()).toHaveValue('Thal, o Errante')
  })

  it('mostra a linhagem já escolhida sob o nome', () => {
    renderStep()

    expect(screen.getByText(/Humano · Guerreiro Nv 2 · Acólito/)).toBeInTheDocument()
  })

  it('ajusta o deslocamento', async () => {
    const { draft } = renderStep()

    await userEvent.click(screen.getByRole('button', { name: /aumentar deslocamento/i }))

    expect(draft.values.displacement).toBe(10)
  })

  it('mostra os PV e PM máximos derivados da construção', () => {
    renderStep()

    // Guerreiro Nv 2, CON 0 → 20 do 1º nível + 5 do 2º. O preview e a ficha
    // têm de dizer o mesmo número.
    expect(screen.getByRole('progressbar', { name: /pontos de vida/i })).toHaveAttribute(
      'aria-valuemax',
      '25',
    )
  })

  it('deixa começar ferido sem mexer no máximo', async () => {
    const { draft } = renderStep()

    const current = screen.getByRole('spinbutton', { name: /pv atual/i })
    await userEvent.clear(current)
    await userEvent.type(current, '9')

    expect(draft.values.hpCurrent).toBe(9)
    expect(draft.values.hpMax).toBe(25)
  })

  // O `max` do input NÃO trava: digitar 99 grava 99 no rascunho e a recusa só
  // chega do servidor, no fim da criação (provado em
  // `api/character_create_http_test.go`). O teste que afirmava o atributo
  // provava fiação e dava a impressão de trava — a garantia mora no servidor, e
  // fechar o vão na tela é decisão de produto, não conserto de teste.

  it('abre a devoção ao escolher um deus e guarda o poder concedido', async () => {
    const { draft } = renderStep((d) => d.setValue('god', 'Khalmyr'))

    const power = screen.getAllByRole('button', { pressed: false })[0]
    await userEvent.click(power)

    expect(draft.values.godPower).toBe(power.textContent?.trim().split('·')[0].trim())
    expect(power).toHaveAttribute('aria-pressed', 'true')
  })

  it('não mostra devoção nenhuma sem deus', () => {
    renderStep()

    expect(screen.queryByText(/devoção a/i)).not.toBeInTheDocument()
  })
})
