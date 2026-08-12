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
import { ClasseStep } from './classe-step'

const OPTIONS = {
  races: ['Anão'],
  classes: ['Arcanista', 'Bárbaro', 'Guerreiro', 'Ladino'],
  origins: ['Artista'],
  gods: [],
  sizes: ['Médio'],
  expertises: [],
} as unknown as CharacterOptions

function renderStep(setup: (draft: CharacterDraftStore) => void = () => {}) {
  const draft = createCharacterDraftStore(new FakeStorage())
  setup(draft)
  render(() => (
    <ForgeProvider draft={draft} options={OPTIONS}>
      <ClasseStep />
    </ForgeProvider>
  ))
  return { draft }
}

const chosen = () => within(screen.getByRole('region', { name: 'Classes escolhidas' }))

describe('ClasseStep — escolher o ofício', () => {
  it('cada ladrilho mostra o que a classe vale no nível 1', () => {
    renderStep()

    expect(screen.getByRole('option', { name: /Guerreiro/ })).toHaveTextContent(/PV \d+/)
  })

  it('a primeira classe escolhida entra como principal, no nível 1', async () => {
    const { draft } = renderStep()

    await userEvent.click(screen.getByRole('option', { name: /Guerreiro/ }))

    expect(draft.values.classes).toEqual([{ className: 'Guerreiro', level: 1 }])
  })

  it('sem classe, convida em vez de ficar em branco', () => {
    renderStep()

    expect(chosen().getByText(/Escolha um ofício/)).toBeInTheDocument()
  })

  it('a classe escolhida ganha o próprio nível e o que ela concede', async () => {
    renderStep((draft) => draft.setValue('classes', [{ className: 'Guerreiro', level: 1 }]))

    expect(chosen().getByLabelText('Nível de Guerreiro')).toHaveValue(1)
    expect(chosen().getByText(/PV \d+ inicial/)).toBeInTheDocument()
  })

  it('digitar o nível não rouba o foco do campo', async () => {
    // A linha é reconstruída a cada tecla (cada edição gera uma entrada nova);
    // se ela for reconciliada por REFERÊNCIA, o campo morre no meio da digitação
    // e o jogador só consegue escrever um dígito.
    renderStep((d) => d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]))
    const field = chosen().getByLabelText('Nível de Guerreiro')

    await userEvent.clear(field)
    await userEvent.type(field, '12')

    expect(field).toHaveFocus()
    expect(field).toHaveValue(12)
  })

  it('mudar o nível grava na classe certa', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [
        { className: 'Guerreiro', level: 1 },
        { className: 'Ladino', level: 1 },
      ]),
    )

    const field = chosen().getByLabelText('Nível de Ladino')
    await userEvent.clear(field)
    await userEvent.type(field, '4')

    expect(draft.values.classes).toEqual([
      { className: 'Guerreiro', level: 1 },
      { className: 'Ladino', level: 4 },
    ])
  })
})

describe('ClasseStep — multiclasse', () => {
  it('a segunda classe pede confirmação antes de entrar', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]),
    )

    await userEvent.click(screen.getByRole('option', { name: /Ladino/ }))

    // Nada entrou ainda: o diálogo explica que não é padrão no nível 1.
    expect(draft.values.classes).toHaveLength(1)
    expect(screen.getByRole('dialog')).toHaveTextContent(/Multiclasse/)
  })

  it('confirmando, a classe entra como adicional', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]),
    )
    await userEvent.click(screen.getByRole('option', { name: /Ladino/ }))

    await userEvent.click(screen.getByRole('button', { name: 'Adicionar mesmo assim' }))

    expect(draft.values.classes).toEqual([
      { className: 'Guerreiro', level: 1 },
      { className: 'Ladino', level: 1 },
    ])
  })

  it('desistindo, nada muda', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]),
    )
    await userEvent.click(screen.getByRole('option', { name: /Ladino/ }))

    await userEvent.click(screen.getByRole('button', { name: 'Cancelar' }))

    expect(draft.values.classes).toHaveLength(1)
  })

  it('tirar uma classe não pede confirmação nenhuma', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [
        { className: 'Guerreiro', level: 1 },
        { className: 'Ladino', level: 1 },
      ]),
    )

    await userEvent.click(screen.getByRole('option', { name: /Ladino/ }))

    expect(draft.values.classes).toEqual([{ className: 'Guerreiro', level: 1 }])
  })

  it('mostra o nível total quando há mais de uma classe', () => {
    renderStep((d) =>
      d.setValue('classes', [
        { className: 'Guerreiro', level: 3 },
        { className: 'Ladino', level: 2 },
      ]),
    )

    expect(screen.getByText(/Nível total 5/)).toBeInTheDocument()
  })
})

describe('ClasseStep — sugestão de atributos', () => {
  it('a primeira classe preenche os atributos sugeridos', async () => {
    const { draft } = renderStep()

    await userEvent.click(screen.getByRole('option', { name: /Guerreiro/ }))

    expect(draft.values.strength).toBeGreaterThan(0)
  })

  it('e diz que fez isso, em vez de mexer em silêncio', async () => {
    renderStep()

    await userEvent.click(screen.getByRole('option', { name: /Guerreiro/ }))

    expect(screen.getByText(/sugest(ã|a)o de Guerreiro/i)).toBeInTheDocument()
  })

  it('desfazer devolve os atributos como estavam', async () => {
    const { draft } = renderStep((d) => d.setValue('charisma', 3))

    await userEvent.click(screen.getByRole('option', { name: /Guerreiro/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Desfazer' }))

    expect(draft.values.charisma).toBe(3)
    expect(draft.values.strength).toBe(0)
  })

  it('a segunda classe NÃO mexe nos atributos — o preset é da principal', async () => {
    const { draft } = renderStep((d) =>
      d.setValue('classes', [{ className: 'Guerreiro', level: 1 }]),
    )
    draft.setValue('intelligence', 2)

    await userEvent.click(screen.getByRole('option', { name: /Arcanista/ }))
    await userEvent.click(screen.getByRole('button', { name: 'Adicionar mesmo assim' }))

    expect(draft.values.intelligence).toBe(2)
    expect(screen.queryByRole('button', { name: 'Desfazer' })).not.toBeInTheDocument()
  })
})
