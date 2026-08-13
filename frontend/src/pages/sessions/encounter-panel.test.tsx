import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { EncounterPanel } from './encounter-panel'

/** Named fake for the session socket — records what the panel pushed. */
class FakeRealtime {
  readonly added: Omit<InitiativeEntry, 'id'>[] = []
  constructor(private readonly entries: InitiativeEntry[] = []) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: this.entries }) as ReturnType<SessionRealtime['state']>,
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      addEntry: (entry: Omit<InitiativeEntry, 'id'>) => {
        this.added.push(entry)
      },
      updateEntry: vi.fn(),
      removeEntry: vi.fn(),
      nextTurn: vi.fn(),
      resetInitiative: vi.fn(),
      populateParty: vi.fn(),
    } as unknown as SessionRealtime
  }
}

function renderPanel(rt: FakeRealtime) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <EncounterPanel rt={rt.asRealtime()} />
    </QueryClientProvider>
  ))
}

beforeEach(() => {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: true,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
  Element.prototype.scrollIntoView = vi.fn()
})
afterEach(() => {
  vi.clearAllMocks()
  document.body.innerHTML = ''
})

describe('EncounterPanel', () => {
  it('o gatilho abre o painel, não manda nada', async () => {
    const rt = new FakeRealtime()
    renderPanel(rt)

    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))

    expect(await screen.findByRole('dialog')).toHaveAccessibleName('Montar encontro')
    expect(rt.added).toHaveLength(0)
  })

  it('não deixa mandar um encontro vazio', async () => {
    renderPanel(new FakeRealtime())
    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))

    expect(screen.getByRole('button', { name: /mandar para a iniciativa/i })).toBeDisabled()
  })

  it('fixa o peek da partida no cabeçalho — o mestre não perde o fio', async () => {
    renderPanel(new FakeRealtime())

    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))

    // O MatchPeek vive no cabeçalho do painel, não atrás dele.
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })
})
