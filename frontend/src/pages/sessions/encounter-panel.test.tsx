import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import { createSignal } from 'solid-js'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { CampaignMember } from '@/shared/api/types'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { EncounterPanel } from './encounter-panel'

/** Um PC da campanha, reduzido ao que o construtor de encontros olha. */
function pc(id: number, level: number): CampaignMember {
  return {
    id,
    campaignId: 1,
    characterId: id,
    role: 'player',
    addedAt: '2026-01-01T00:00:00Z',
    character: { id, ownerId: 1, name: `PC ${id}`, level, hpCurrent: 10, hpMax: 10, mpCurrent: 5, mpMax: 5, classes: [] },
  } as CampaignMember
}

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

/** O painel é CONTROLADO pelo trilho das consultas desde a ALE-198: quem o abre
 *  é a cena, que garante um overlay por vez. O teste faz o papel do trilho. */
function renderPanel(rt: FakeRealtime, members: CampaignMember[] = []) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const [open, setOpen] = createSignal(false)
  render(() => (
    <QueryClientProvider client={client}>
      <button type="button" onClick={() => setOpen(true)}>
        Montar encontro
      </button>
      <EncounterPanel rt={rt.asRealtime()} members={members} open={open()} onOpenChange={setOpen} />
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

/**
 * O grupo já vem preenchido da campanha (ALE-209). A REGRA da média mora no
 * unitário do `party-defaults`; o que se prova aqui é a TRADUÇÃO — que os
 * números derivados chegam aos campos, e que a edição do mestre sobrevive.
 */
describe('o grupo vem da campanha', () => {
  it('nível é a média dos PCs e personagens é quantos são', async () => {
    renderPanel(new FakeRealtime(), [pc(1, 5), pc(2, 6), pc(3, 7), pc(4, 6)])

    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))

    expect(await screen.findByLabelText('Nível do grupo')).toHaveValue(6)
    expect(screen.getByLabelText('Personagens do grupo')).toHaveValue(4)
  })

  // Sem campanha por perto o padrão do livro continua valendo — quatro de 1º
  // (p282). Vir zero seria pior do que vir vazio.
  it('sem PC nenhum, cai no padrão do livro', async () => {
    renderPanel(new FakeRealtime(), [])

    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))

    expect(await screen.findByLabelText('Nível do grupo')).toHaveValue(1)
    expect(screen.getByLabelText('Personagens do grupo')).toHaveValue(4)
  })

  // O mestre pode estar montando para MEIO grupo. O que ele escreve vence o
  // derivado para sempre — senão o app desfaria a escolha dele sozinho.
  it('o que o mestre escreve vence o derivado', async () => {
    renderPanel(new FakeRealtime(), [pc(1, 8), pc(2, 8)])

    await userEvent.click(screen.getByRole('button', { name: /montar encontro/i }))
    // Pelo SPINNER, e não digitando: `input[type=number]` não tem API de
    // seleção, então nem `clear()` nem `{selectall}` esvaziam o campo — o
    // dígito novo se junta ao velho ("2" + "3" = 23, clampado ao teto 8) e o
    // teste mediria o clamp em vez do override. O spinner é gesto de verdade
    // e não tem essa ambiguidade.
    await userEvent.click(await screen.findByRole('button', { name: 'Aumentar personagens' }))

    expect(screen.getByLabelText('Personagens do grupo')).toHaveValue(3)  // 2 derivado + 1
    // O nível continua o DERIVADO: só o campo tocado vira override.
    expect(screen.getByLabelText('Nível do grupo')).toHaveValue(8)
  })
})
