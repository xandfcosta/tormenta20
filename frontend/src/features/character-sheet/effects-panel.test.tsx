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


function renderPanel(char: Character = makeCharacter()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(characterQueryOptions(char.id).queryKey, char)
  const powerUses = createPowerUsesStore(new FakeStorage())
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <PowerUsesProvider store={powerUses}>
          <EffectsPanel character={char} />
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

  it('aplicar condição pelo picker manda a lista somada', async () => {
    const api = await import('@/shared/api/api')
    const update = vi
      .spyOn(api.api.characters, 'updateConditions')
      .mockResolvedValue({ activeConditions: '["cego"]' })
    const { user } = renderPanel()

    await user.click(screen.getByRole('combobox', { name: 'Aplicar condição' }))
    await user.click(await screen.findByRole('option', { name: 'Cego' }))

    await waitFor(() => expect(update).toHaveBeenCalledWith(1, ['cego']))
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
