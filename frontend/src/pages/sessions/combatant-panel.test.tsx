import { render, screen } from '@solidjs/testing-library'
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { makeCharacter } from '@/entities/character/__fixtures__/character'
import { characterQueryOptions } from '@/entities/character/queries'
import { campaignCreaturesQueryOptions } from '@/entities/creature/queries'
import type { InitiativeEntry } from '@/shared/realtime/realtime'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { createConditionalsStore } from '@/shared/stores/conditionals-store'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { createPowerUsesStore } from '@/shared/stores/power-uses-store'
import { FakeStorage } from '@/shared/test/fake-storage'
import { CombatantPanel } from './combatant-panel'

/**
 * O COMBATENTE ABERTO (ALE-197, grupo C).
 *
 * O painel decide uma coisa só, e ela é a razão de ele existir: PC e NPC são
 * combatentes diferentes. O PC tem ficha atrás dele e NÃO ganha cabeçalho —
 * a faixa já diz o nome, e repeti-lo custava 61px, quase 40% da região no
 * celular deitado (ALE-145). O NPC não tem ficha, então o nome dele precisa de
 * um cabeçalho próprio.
 *
 * Essa bifurcação nunca tinha sido montada. Ela é `keyed` de propósito: trocar
 * de combatente tem de RECONSTRUIR o cartão, e um `Show` sem `keyed` reusaria o
 * nó — a armadilha nº2 do guia do front.
 */

function linha(overrides: Partial<InitiativeEntry> = {}): InitiativeEntry {
  return { id: 'e1', label: 'Ogro Brutamontes', initiative: 12, type: 'npc', ...overrides }
}

function renderPanel(entry: InitiativeEntry, onClose = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(campaignCreaturesQueryOptions(1).queryKey, [])
  if (entry.characterId) {
    client.setQueryData(
      characterQueryOptions(entry.characterId).queryKey,
      makeCharacter({ id: entry.characterId, name: 'Arwen' }),
    )
  }
  render(() => (
    <QueryClientProvider client={client}>
      <ConditionalsProvider store={createConditionalsStore(new FakeStorage())}>
        <PowerUsesProvider store={createPowerUsesStore(new FakeStorage())}>
          <CombatantPanel entry={entry} onClose={onClose} campaignId={1} />
        </PowerUsesProvider>
      </ConditionalsProvider>
    </QueryClientProvider>
  ))
  return { user: userEvent.setup(), onClose }
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

describe('CombatantPanel', () => {
  it('o NPC ganha cabeçalho com o nome dele', async () => {
    renderPanel(linha())

    // O NPC não tem faixa de ficha: sem este cabeçalho o mestre abriria um
    // painel que não diz de quem é.
    expect(await screen.findByRole('heading', { name: 'Ogro Brutamontes' })).toBeInTheDocument()
    // E não tem ficha nenhuma para mostrar — NPC não tem personagem atrás.
    expect(screen.queryByText('Perícias')).not.toBeInTheDocument()
  })

  it('o PC traz a ficha, e o nome dele aparece uma vez só', async () => {
    renderPanel(linha({ label: 'Arwen', type: 'character', characterId: 7 }))

    await screen.findByLabelText('Fechar o combatente')

    // O PC traz a FICHA atrás dele — é ela que responde "quanto de Percepção?"
    // sem o mestre sair da mesa.
    expect(await screen.findByText('Perícias')).toBeInTheDocument()
    // E o nome aparece UMA vez: repeti-lo num cabeçalho acima da faixa custava
    // 61px numa região de 165px no celular deitado (ALE-145).
    expect(screen.getAllByRole('heading', { name: 'Arwen' })).toHaveLength(1)
  })

  it('fechar avisa quem abriu', async () => {
    const { user, onClose } = renderPanel(linha())

    await user.click(await screen.findByLabelText('Fechar o combatente'))

    expect(onClose).toHaveBeenCalledOnce()
  })

  it('sem `onApplyEffect`, o seletor de efeito não aparece', async () => {
    renderPanel(linha())

    await screen.findByLabelText('Fechar o combatente')
    // A prop é opcional porque a mesa do jogador abre o mesmo painel sem poder
    // aplicar buff — oferecer o seletor ali seria oferecer um comando do mestre.
    expect(screen.queryByLabelText(/Aplicar efeito/)).not.toBeInTheDocument()
  })
})
