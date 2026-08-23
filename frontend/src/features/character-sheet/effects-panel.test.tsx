import { FakeStorage } from '@/shared/test/fake-storage'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor, within } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { characterQueryOptions } from '@/entities/character/queries'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import type { ActiveEffect, Character } from '@/shared/api/api'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { createPowerUsesStore } from '@/shared/stores/power-uses-store'
import { EffectsPanel } from './effects-panel'

function effect(overrides: Partial<ActiveEffect> = {}): ActiveEffect {
  return {
    id: 1,
    catalogId: 'pocao-de-cura',
    scope: 'scene',
    modifiers: '[]',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}


function renderPanel(char: Character = makeCharacter(), inSession = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  const powerUses = createPowerUsesStore(new FakeStorage())
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <PowerUsesProvider store={powerUses}>
          <EffectsPanel character={char} inSession={inSession} />
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), client, powerUses }
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: false,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
})

afterEach(() => {
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('EffectsPanel — condições do livro', () => {
  it('mostra a condição ativa com o efeito que ela aplica', () => {
    renderPanel(makeCharacter({ activeConditions: '["caido"]' }))

    expect(screen.getByText('Caído')).toBeInTheDocument()
    // ALE-28: a condição tem de dizer o que faz, não ser só um badge.
    expect(screen.getByRole('button', { name: 'Remover condição Caído' })).toBeInTheDocument()
  })

  it('remover a condição escreve a lista sem ela', async () => {
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateConditions')
      .mockResolvedValue({ activeConditions: '[]' })
    const { user } = renderPanel(makeCharacter({ activeConditions: '["caido"]' }))

    await user.click(screen.getByRole('button', { name: 'Remover condição Caído' }))

    expect(update).toHaveBeenCalledWith(1, [])
  })

  // ALE-216: era um campo de busca embutido; agora é o MESMO gesto do "Aplicar
  // efeito" ao lado — botão de adicionar que abre um diálogo.
  it('aplicar condição pelo diálogo manda a lista somada', async () => {
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateConditions')
      .mockResolvedValue({ activeConditions: '["cego"]' })
    const { user } = renderPanel()

    await user.click(screen.getByRole('button', { name: 'Aplicar condição' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /^Cego/ }))

    await waitFor(() => expect(update).toHaveBeenCalledWith(1, ['cego']))
  })

  // A metade que a ALE-216 avisa: a ação acontece DENTRO do modal, e um toast
  // disparado dali nunca é anunciado (o modal marca os irmãos `aria-hidden`).
  // O diálogo fica aberto com a falha escrita nele.
  it('falha ao aplicar é dita INLINE, dentro do diálogo aberto', async () => {
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'updateConditions').mockRejectedValue(new Error('500'))
    const { user } = renderPanel()

    await user.click(screen.getByRole('button', { name: 'Aplicar condição' }))
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: /^Cego/ }))

    const alerta = await within(dialog).findByRole('alert')
    expect(alerta).toHaveTextContent(/Não foi possível aplicar a condição/)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})

describe('EffectsPanel — efeitos ativos', () => {
  it('lista o efeito com o escopo em que ele expira', () => {
    renderPanel(makeCharacter({ activeEffects: [effect({ scope: 'day' })] }))

    expect(screen.getByText('dia')).toBeInTheDocument()
  })

  it('sem consumível ativo, explica de onde eles vêm', () => {
    renderPanel()
    expect(screen.getByText(/Nenhum consumível ativo/)).toBeInTheDocument()
  })

  it('encerrar cena limpa os efeitos de cena e zera os usos por cena', async () => {
    const api = await import('@/shared/api/api')
    vi.spyOn(api.api.characters, 'endScene').mockResolvedValue({ clearedScopes: ['scene'] })
    const { user, client, powerUses } = renderPanel(
      makeCharacter({ activeEffects: [effect({ id: 1, scope: 'scene' }), effect({ id: 2, scope: 'day' })] }),
    )
    powerUses.bump(1, 'class.bardo.inspiracao', 'scene')

    await user.click(screen.getByRole('button', { name: 'Encerrar cena' }))
    // O gatilho e o botão de confirmar têm o MESMO rótulo — busca dentro do
    // diálogo, senão o clique volta pro gatilho e nada é encerrado.
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Encerrar cena' }))

    await waitFor(() => {
      const cached = client.getQueryData<Character>(characterQueryOptions(1).queryKey)
      expect(cached?.activeEffects.map((e) => e.id)).toEqual([2])
    })
    // O limite de uso do livro acompanha a mesma fronteira de cena.
    expect(powerUses.used(1, 'class.bardo.inspiracao').scene).toBe(0)
  })
})

describe('EffectsPanel — situação', () => {
  it('sem condicional nenhum, ensina como conseguir um', () => {
    renderPanel()
    expect(screen.getByText(/Nenhum efeito condicional disponível/)).toBeInTheDocument()
  })
})

// ALE-216: dentro da sessão o descanso é decisão da MESA — encerrar cena e
// encerrar dia são do mestre, que tem os dois no menu da sessão. Esconder é só
// UX; a recusa que vale é a do handler Go (character_effects_http_test.go).
describe('EffectsPanel — encerrar cena/dia na sessão', () => {
  it('fora da sessão o jogador administra a própria ficha', () => {
    renderPanel()

    expect(screen.getByRole('button', { name: 'Encerrar cena' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Encerrar dia' })).toBeInTheDocument()
  })

  it('numa sessão ao vivo as duas ações somem', () => {
    renderPanel(makeCharacter(), true)

    expect(screen.queryByRole('button', { name: 'Encerrar cena' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Encerrar dia' })).not.toBeInTheDocument()
    // O que continua sendo dele: aplicar efeito e condição na própria ficha.
    expect(screen.getByRole('button', { name: 'Aplicar efeito' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Aplicar condição' })).toBeInTheDocument()
  })
})
