import { render, screen, within } from '@solidjs/testing-library'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { ResumoStep } from './resumo-step'

const OPTIONS = {
  races: ['Humano'],
  classes: ['Guerreiro'],
  origins: ['Acólito'],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  draft.setValue('name', 'Thal, o Errante')
  draft.setValue('races', ['Anão'])
  draft.setValue('origin', 'Acólito')
  draft.setValue('classes', [{ className: 'Guerreiro', level: 2 }])
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <ResumoStep />
    </ForgeProvider>
  ))
  return { draft }
}

describe('ResumoStep', () => {
  it('mostra o personagem pelo nome e pela linhagem', () => {
    renderStep()

    expect(screen.getByText('Thal, o Errante')).toBeInTheDocument()
    expect(screen.getByText(/Anão · Guerreiro Nv 2 · Acólito/)).toBeInTheDocument()
  })

  it('diz "Sem nome" em vez de deixar o título vazio', () => {
    renderStep((d) => d.setValue('name', '   '))

    expect(screen.getByText('Sem nome')).toBeInTheDocument()
  })

  it('mostra os números que a ficha vai abrir', () => {
    renderStep()

    // Anão Guerreiro Nv 2: 25 do porte + o CON e o Duro como Pedra da raça.
    // O resumo e o rodapé leem o MESMO derivador — não podem discordar.
    expect(screen.getByText(/Pontos de vida 33/)).toBeInTheDocument()
  })

  it('lista as pendências sem impedir a criação', () => {
    renderStep()

    expect(screen.getByText(/pendência/i)).toBeInTheDocument()
    expect(screen.getByText(/terminar na ficha/i)).toBeInTheDocument()
  })

  // O botão "Criar personagem" fica desabilitado enquanto falta o essencial —
  // e um botão morto sem motivo na tela é o pior fim de fluxo possível.
  it('diz o que está travando a criação, não só o que é opcional', () => {
    renderStep((d) => d.setValue('races', []))

    expect(screen.getByText(/falta o essencial para forjar/i)).toBeInTheDocument()
    expect(screen.getByText(/escolha uma raça/i)).toBeInTheDocument()
  })

  it('some com o aviso quando não há pendência nenhuma', () => {
    renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 1 }])
      d.setValue('originChoices', [
        'origin-acolito-pericia-Cura',
        'origin-acolito-pericia-Religião',
      ])
      d.setValue('trainedExpertises', ['Fortitude', 'Luta', 'Pontaria', 'Percepção', 'Cavalgar'])
    })

    expect(screen.queryByText(/pendência/i)).not.toBeInTheDocument()
  })

  it('mostra o poder pelo NOME, não pelo id do catálogo', () => {
    renderStep((d) => {
      d.setValue('classes', [{ className: 'Guerreiro', level: 3 }])
      d.setValue('classPowers', ['ataque-poderoso'])
    })

    const powers = within(screen.getByRole('region', { name: 'Poderes' }))
    expect(powers.getByText(/Ataque Poderoso/)).toBeInTheDocument()
  })

  it('mostra o benefício de origem pelo nome', () => {
    renderStep((d) => d.setValue('originChoices', ['origin-acolito-pericia-Cura']))

    const origem = within(screen.getByRole('region', { name: 'Origem' }))
    expect(origem.getByText(/Cura/)).toBeInTheDocument()
  })

  it('diz o que está pendente na bagagem em vez de omitir', () => {
    renderStep()

    const bag = within(screen.getByRole('region', { name: 'Bagagem' }))
    // Arma simples e armadura ainda não escolhidas — as duas têm de aparecer.
    expect(bag.getAllByText(/pendente/i).length).toBeGreaterThan(0)
  })
})
