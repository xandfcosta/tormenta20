import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { PoderesStep } from './poderes-step'

const OPTIONS = {
  races: ['Humano'],
  classes: ['Guerreiro', 'Arcanista'],
  origins: [],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  draft.setValue('races', ['Humano'])
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <PoderesStep />
    </ForgeProvider>
  ))
  return { draft }
}

const taken = () => within(screen.getByRole('region', { name: 'Seus poderes' }))

describe('PoderesStep', () => {
  it('pede uma classe antes de qualquer poder', () => {
    renderStep()

    expect(screen.getByText(/escolha uma classe primeiro/i)).toBeInTheDocument()
  })

  it('diz que ainda não há vaga, em vez de mostrar uma lista vazia', () => {
    renderStep((d) => d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]))

    expect(screen.getByText(/nenhuma vaga de poder ainda/i)).toBeInTheDocument()
  })

  // Guerreiro 1 + Ladino 1 é um personagem de nível 2 que ainda não ganha
  // poder: a vaga é do 2º nível DE UMA CLASSE, não do total.
  it('não promete vaga para multiclasse de dois níveis 1', () => {
    renderStep((d) =>
      d.setValue('classes', [
        { className: 'Guerreiro', level: 1 },
        { className: 'Ladino', level: 1 },
      ]),
    )

    expect(screen.getByText(/nenhuma vaga de poder ainda/i)).toBeInTheDocument()
  })

  it('conta as vagas que os níveis rendem', () => {
    renderStep((d) => d.setValue('classes', [{ className: 'Guerreiro', level: 3 }]))

    // Nv 3 → duas vagas (2º e 3º nível).
    expect(taken().getByText('0 de 2')).toBeInTheDocument()
  })

  it('mostra uma vaga livre por slot ainda não gasto', () => {
    renderStep((d) => d.setValue('classes', [{ className: 'Guerreiro', level: 3 }]))

    expect(taken().getAllByText(/vaga livre/i)).toHaveLength(2)
  })

  it('leva o poder escolhido para a coluna da direita', () => {
    renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 3 }])
      d.setValue('classPowers', ['ataque-poderoso'])
    })

    expect(taken().getByText('Ataque Poderoso')).toBeInTheDocument()
    expect(taken().getByText('1 de 2')).toBeInTheDocument()
    expect(taken().getAllByText(/vaga livre/i)).toHaveLength(1)
  })

  it('remover devolve a vaga', async () => {
    const { draft } = renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 3 }])
      d.setValue('classPowers', ['ataque-poderoso'])
    })

    await userEvent.click(taken().getByRole('button', { name: /remover ataque poderoso/i }))

    expect(draft.values.classPowers).toEqual([])
    expect(taken().getAllByText(/vaga livre/i)).toHaveLength(2)
  })

  // Trocar de classe depois de escolher poderes deixava a cota dizendo "2 de 2"
  // sobre uma coluna vazia: o poder da classe antiga sumia da lista mas
  // continuava gastando a vaga, e não havia como removê-lo.
  it('mostra o poder que saiu da lista da classe, com saída', async () => {
    const { draft } = renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 3 }])
      d.setValue('classPowers', ['poder-de-outra-classe'])
    })

    expect(taken().getByText(/ainda ocupa uma vaga/i)).toBeInTheDocument()
    await userEvent.click(taken().getByRole('button', { name: /remover poder-de-outra-classe/i }))

    expect(draft.values.classPowers).toEqual([])
  })

  it('a busca do catálogo não some com a coluna dos escolhidos', async () => {
    renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 3 }])
      d.setValue('classPowers', ['ataque-poderoso'])
    })

    await userEvent.type(screen.getByRole('searchbox', { name: /buscar poder/i }), 'zzz')

    expect(screen.getByText(/nenhum poder corresponde/i)).toBeInTheDocument()
    expect(taken().getByText('Ataque Poderoso')).toBeInTheDocument()
  })

  it('oferece o caminho quando a classe e o nível pedem', () => {
    renderStep((d) => d.setValue('classes', [{ className: 'Arcanista', level: 1 }]))

    expect(screen.getByRole('button', { name: /caminho de arcanista/i })).toBeInTheDocument()
  })

  it('não oferece caminho para uma classe que não tem', () => {
    renderStep((d) => d.setValue('classes', [{ className: 'Guerreiro', level: 3 }]))

    expect(screen.queryByRole('button', { name: /caminho de/i })).not.toBeInTheDocument()
  })
})
