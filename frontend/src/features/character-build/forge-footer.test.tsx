import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { FakeStorage } from '@/shared/test/fake-storage'
import {
  type CharacterDraftStore,
  createCharacterDraftStore,
} from '@/shared/stores/character-draft-store'
import { ForgeFooter } from './forge-footer'

function renderFooter(
  setup: (draft: CharacterDraftStore) => void = () => {},
  props: Partial<Parameters<typeof ForgeFooter>[0]> = {},
) {
  const draft = createCharacterDraftStore(new FakeStorage())
  setup(draft)
  const onStep = vi.fn()
  const onCreate = vi.fn()
  render(() => (
    <ForgeFooter
      draft={draft}
      current="raca"
      submitting={false}
      onStep={onStep}
      onCreate={onCreate}
      {...props}
    />
  ))
  return { draft, onStep, onCreate }
}

/** Lê o número de uma estatística pela linha falada ("Defesa 12"). */
const statValue = (label: string): number => {
  const spoken = screen.getByText(new RegExp(`^${label} \\d+$`))
  return Number(spoken.textContent?.replace(`${label} `, ''))
}

const anao = (draft: CharacterDraftStore) => {
  draft.setValue('races', ['Anão'])
  draft.setValue('classes', [{ className: 'Guerreiro', level: 1 }])
}

describe('ForgeFooter', () => {
  it('mostra um convite enquanto nada foi escolhido', () => {
    renderFooter()

    expect(screen.getByText('Novo personagem')).toBeInTheDocument()
  })

  it('resume a identidade escolhida até aqui', () => {
    renderFooter((draft) => {
      anao(draft)
      draft.setValue('name', 'Aknor')
    })

    expect(screen.getByText('Aknor')).toBeInTheDocument()
    expect(screen.getByText(/Anão · Guerreiro Nv 1/)).toBeInTheDocument()
  })

  it('pinta DEF/PV/PM derivados da escolha, sem ninguém digitar', () => {
    renderFooter(anao)

    // Guerreiro Nv1: PV e DEF saem do motor, ninguém digita.
    expect(statValue('Defesa')).toBeGreaterThan(0)
    expect(statValue('Pontos de vida')).toBeGreaterThan(0)
  })

  it('o número se move quando a escolha muda', async () => {
    const { draft } = renderFooter(anao)
    const before = statValue('Pontos de vida')

    draft.setValue('constitution', 3)

    expect(statValue('Pontos de vida')).toBeGreaterThan(before)
  })

  it('com multiclasse, mostra as duas classes e o nível TOTAL', () => {
    renderFooter((draft) =>
      draft.setValue('classes', [
        { className: 'Guerreiro', level: 3 },
        { className: 'Ladino', level: 2 },
      ]),
    )

    expect(screen.getByText(/Guerreiro · Ladino Nv 5/)).toBeInTheDocument()
  })

  it('Próximo fica trancado enquanto o passo não está pronto', async () => {
    const { onStep } = renderFooter()

    const next = screen.getByRole('button', { name: /Próximo/ })
    expect(next).toBeDisabled()
    await userEvent.click(next)

    expect(onStep).not.toHaveBeenCalled()
  })

  it('Próximo destranca assim que o passo se resolve', async () => {
    const { onStep } = renderFooter(anao)

    await userEvent.click(screen.getByRole('button', { name: /Próximo/ }))

    expect(onStep).toHaveBeenCalledWith(1)
  })

  it('no primeiro passo não há Voltar', () => {
    renderFooter(anao)

    expect(screen.queryByRole('button', { name: /Voltar/ })).not.toBeInTheDocument()
  })

  it('a partir do segundo passo, Voltar recua um', async () => {
    const { onStep } = renderFooter(anao, { current: 'classe' })

    await userEvent.click(screen.getByRole('button', { name: /Voltar/ }))

    expect(onStep).toHaveBeenCalledWith(-1)
  })

  /**
   * O único lugar onde a recusa do servidor aparece na criação: o rodapé.
   * A prop `error` existia e nenhum teste passava uma — quem apagasse o `Show`
   * levaria o jogador a clicar "Criar" e não ver nada acontecer.
   */
  it('a recusa do servidor aparece anunciada como alerta', () => {
    renderFooter(() => {}, { error: 'HP current cannot exceed HP max' })

    const alerta = screen.getByRole('alert')
    expect(alerta).toHaveTextContent('HP current cannot exceed HP max')
  })

  it('sem erro, não fica um alerta vazio na tela', () => {
    renderFooter()

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('no Resumo a ação vira Criar', async () => {
    const { onCreate } = renderFooter(
      (draft) => {
        anao(draft)
        draft.setValue('name', 'Aknor')
        draft.setValue('origin', 'Artista')
      },
      { current: 'resumo' },
    )

    await userEvent.click(screen.getByRole('button', { name: /Criar personagem/ }))

    expect(onCreate).toHaveBeenCalled()
  })

  it('Criar fica trancado enquanto falta um passo', () => {
    renderFooter(anao, { current: 'resumo' }) // sem nome nem origem

    expect(screen.getByRole('button', { name: /Criar personagem/ })).toBeDisabled()
  })

  it('enquanto salva, o botão diz o que está acontecendo e não aceita clique', () => {
    renderFooter(
      (draft) => {
        anao(draft)
        draft.setValue('name', 'Aknor')
        draft.setValue('origin', 'Artista')
      },
      { current: 'resumo', submitting: true },
    )

    expect(screen.getByRole('button', { name: /Forjando/ })).toBeDisabled()
  })
})
