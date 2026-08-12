import { render, screen, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { ForgeProvider } from '@/features/character-build/forge-context'
import { periciaPlan } from '@/features/character-build/pericia-helpers'
import type { CharacterOptions } from '@/shared/api/api'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { PericiasStep } from './pericias-step'

const OPTIONS = {
  races: ['Anão'],
  classes: ['Guerreiro'],
  origins: [],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <PericiasStep />
    </ForgeProvider>
  ))
  return { draft }
}

const guerreiro = (intMod = 0) => {
  const plan = periciaPlan('Guerreiro', intMod, [])
  if (!plan) throw new Error('Guerreiro sem plano de perícias')
  return plan
}

const withClass = (intelligence = 0) => (draft: CharacterDraftStore) => {
  draft.setValue('classes', [{ className: 'Guerreiro', level: 1 }])
  draft.setValue('intelligence', intelligence)
}

const classBand = () => within(screen.getByRole('region', { name: 'Perícias · Da classe' }))
const freeBand = () => within(screen.getByRole('region', { name: 'Perícias · Livre' }))

describe('PericiasStep', () => {
  it('sem classe, diz o que falta em vez de mostrar uma lista vazia', () => {
    renderStep()

    expect(screen.getByText(/Escolha uma classe primeiro/)).toBeInTheDocument()
  })

  it('as perícias fixas da classe entram sozinhas — não são escolha', () => {
    const { draft } = renderStep(withClass())

    for (const fixed of guerreiro().fixed) {
      expect(draft.values.trainedExpertises).toContain(fixed)
    }
  })

  it('a cota da classe conta o que já foi marcado', async () => {
    renderStep(withClass())

    await userEvent.click(classBand().getByRole('button', { name: guerreiro().classPool[0] }))

    expect(classBand().getByText(`1 de ${guerreiro().classCount}`)).toBeInTheDocument()
  })

  it('marcar de novo desmarca', async () => {
    const { draft } = renderStep(withClass())
    const first = guerreiro().classPool[0]

    await userEvent.click(classBand().getByRole('button', { name: first }))
    await userEvent.click(classBand().getByRole('button', { name: first }))

    expect(draft.values.trainedExpertises).not.toContain(first)
  })

  it('sem Inteligência não há banda livre', () => {
    renderStep(withClass(0))

    expect(screen.queryByRole('region', { name: 'Perícias · Livre' })).not.toBeInTheDocument()
  })

  it('Inteligência abre a banda livre e diz de onde ela vem', () => {
    renderStep(withClass(2))

    expect(freeBand().getByText(/2 de Inteligência/)).toBeInTheDocument()
  })

  it('a banda livre tranca quando a cota acaba', async () => {
    const plan = guerreiro(1)
    renderStep((draft) => {
      withClass(1)(draft)
      draft.setValue('trainedExpertises', [plan.freePool[0]])
    })

    expect(freeBand().getByRole('button', { name: plan.freePool[1] })).toBeDisabled()
    // A já marcada continua clicável — senão não dá para trocar de ideia.
    expect(freeBand().getByRole('button', { name: plan.freePool[0] })).toBeEnabled()
  })

  it('explica o transbordo quando a escolha da classe passa a gastar ponto livre', async () => {
    const plan = guerreiro(2)
    renderStep((draft) => {
      withClass(2)(draft)
      draft.setValue('trainedExpertises', plan.classPool.slice(0, plan.classCount))
    })

    expect(screen.queryByText(/usando uma perícia livre/)).not.toBeInTheDocument()

    await userEvent.click(
      classBand().getByRole('button', { name: plan.classPool[plan.classCount] }),
    )

    expect(screen.getByText(/usando uma perícia livre/)).toBeInTheDocument()
  })

  it('diz quantas perícias ainda faltam', () => {
    renderStep(withClass())

    expect(screen.getByText(/Faltam \d+ perícias/)).toBeInTheDocument()
  })

  it('o aviso some quando tudo foi escolhido', () => {
    const plan = guerreiro(0)
    renderStep((draft) => {
      withClass(0)(draft)
      draft.setValue('trainedExpertises', [
        ...plan.fixed,
        ...plan.classPool.slice(0, plan.classCount),
      ])
    })

    expect(screen.queryByText(/Faltam/)).not.toBeInTheDocument()
  })
})
