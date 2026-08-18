import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/solid-router'
import { createSignal } from 'solid-js'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@/shared/api/api'
import type { BoardState } from '@/shared/realtime/realtime'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { SessionGmView } from './session-gm-view'

/**
 * As regiões da cena do mestre (ALE-130).
 *
 * O defeito que motivou este arquivo: entre 1024 e 1536 o seletor oferecia três
 * opções e DUAS desenhavam a mesma tela — "combate" e "mesa" eram idênticas,
 * porque nessa faixa a iniciativa aparece sempre e só a segunda coluna alterna.
 *
 * O que se prova aqui é a TROCA DE CONTEÚDO da segunda coluna. Um teste que
 * afirmasse "a iniciativa está na tela" passaria verde nos dois casos — foi
 * exatamente esse tipo de asserção que deixou o defeito passar.
 */
class FakeRealtime {
  private readonly live = createSignal<BoardState | null>(null)

  /** Abre o tabuleiro DEPOIS da montagem, que é como ele chega: pelo socket. */
  abrirTabuleiro() {
    this.live[1]({ version: 1, place: 'Cripta', terrain: 'pedra', tokens: [] })
  }

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: [], round: 1, turnIndex: -1 }),
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      restFlash: () => null,
      board: this.live[0],
      rest: vi.fn(),
      nextTurn: vi.fn(),
      previousTurn: vi.fn(),
      resetInitiative: vi.fn(),
      populateParty: vi.fn(),
      openBoard: vi.fn(),
    } as unknown as SessionRealtime
  }
}

const SESSAO = { id: 4, campaignId: 1, sessionNumber: 4, status: 'active' } as unknown as Session

/** Simula uma faixa de largura: cada media query responde o que o teste mandar. */
function comLargura(faixa: { sideBySide: boolean; threeUp: boolean }) {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    matches: media.includes('1536') ? faixa.threeUp : faixa.sideBySide,
    media,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })) as unknown as typeof window.matchMedia
}

function renderCena(rt: FakeRealtime = new FakeRealtime()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Router de memória porque a cena tem links (sair da sessão, ficha) — mesmo
  // arreio do `match-shell.test.tsx`.
  const root = createRootRoute({
    component: () => (
      <QueryClientProvider client={client}>
        <SessionGmView
          campaignId={1}
          sessionId={4}
          session={SESSAO}
          rt={rt.asRealtime()}
          myCharacterIds={new Set<number>()}
        />
      </QueryClientProvider>
    ),
  })
  const router = createRouter({ routeTree: root, history: createMemoryHistory() })
  render(() => <RouterProvider router={router} />)
  return userEvent.setup()
}

beforeEach(() => comLargura({ sideBySide: true, threeUp: false }))

describe('as regiões da cena do mestre em duas colunas', () => {
  // Com as duas colunas na tela não existe seletor de região: a iniciativa é a
  // espinha e fica sempre visível. Quem troca é a barra de abas da mesa, que
  // fica EXATAMENTE sobre a coluna que ela troca — uma faixa atravessando a tela
  // inteira para mudar só a direita é um controle desalinhado do efeito.
  //
  // `findBy` na primeira busca: o RouterProvider monta a rota num microtask, e
  // uma busca síncrona olharia a árvore antes de a cena existir.
  it('não há seletor de região; o tabuleiro é aba da mesa', async () => {
    renderCena()

    expect(await screen.findByRole('tab', { name: /Tabuleiro/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'combate' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'mesa' })).not.toBeInTheDocument()
  })

  it('a aba do tabuleiro TROCA o conteúdo da coluna da direita', async () => {
    const user = renderCena()

    expect(await screen.findByRole('tab', { name: /Combatente/ })).toBeInTheDocument()
    expect(screen.queryByText(/Nenhum tabuleiro aberto/)).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: /Tabuleiro/ }))

    expect(screen.getByText(/Nenhum tabuleiro aberto/)).toBeInTheDocument()
    // E a iniciativa continua na tela: ela é a espinha da cena.
    expect(screen.getByRole('heading', { name: 'Iniciativa' })).toBeInTheDocument()
  })
})

describe('as regiões da cena do mestre em três colunas', () => {
  beforeEach(() => comLargura({ sideBySide: true, threeUp: true }))

  // A partir de 1536 o tabuleiro tem COLUNA PRÓPRIA, e aí ele não pode ser
  // também aba: seriam duas cópias do mesmo tabuleiro na mesma tela, cada uma
  // com sua janela. A sabotagem que deixava a aba sempre visível passava verde
  // antes deste teste existir.
  it('o tabuleiro tem coluna e NÃO é aba', async () => {
    renderCena()

    expect(await screen.findByRole('tab', { name: /Combatente/ })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: /Tabuleiro/ })).not.toBeInTheDocument()
    // Ele está na tela por conta própria, ao lado da mesa.
    expect(screen.getByText(/Nenhum tabuleiro aberto/)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Iniciativa' })).toBeInTheDocument()
  })
})

describe('as regiões da cena do mestre numa coluna só', () => {
  beforeEach(() => comLargura({ sideBySide: false, threeUp: false }))

  // Abaixo de 1024 cabe UMA região por vez, e aí "combate" é uma escolha de
  // verdade: é a única forma de ver a iniciativa. O tabuleiro continua sendo
  // aba da mesa, não uma terceira região.
  it('o seletor tem duas regiões, e o combate esconde a mesa', async () => {
    const user = renderCena()

    expect(await screen.findByRole('button', { name: 'combate' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'mesa' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'combate' }))

    expect(screen.getByRole('heading', { name: 'Iniciativa' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: /Combatente/ })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'mesa' }))

    expect(screen.getByRole('tab', { name: /Tabuleiro/ })).toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Iniciativa' })).not.toBeInTheDocument()
  })
})

/**
 * Abrir o tabuleiro tem de MOSTRAR o tabuleiro (ALE-161).
 *
 * Abaixo de 1536 ele não tem coluna própria: é aba da mesa. E a aba ativa
 * continuava sendo "combatente", então o mestre num laptop de 1440 abria o
 * tabuleiro e via 830×745px dizendo "clique no nome de um combatente" — o
 * tabuleiro existia, com peças, e não aparecia em lugar nenhum.
 *
 * O gatilho é a TRANSIÇÃO (o tabuleiro chega pelo socket), não o estado: reagir
 * ao estado brigaria com o mestre que foi de propósito ao bestiário.
 */
describe('abrir o tabuleiro abaixo de 1536', () => {
  it('traz a aba do tabuleiro para a frente', async () => {
    const rt = new FakeRealtime()
    renderCena(rt)
    await screen.findByRole('tab', { name: /Tabuleiro/ })
    // Antes de abrir, quem manda é o combatente.
    expect(screen.getByText(/Clique no nome de um combatente/)).toBeInTheDocument()

    rt.abrirTabuleiro()

    expect(await screen.findByText('Cripta')).toBeInTheDocument()
    expect(screen.queryByText(/Clique no nome de um combatente/)).not.toBeInTheDocument()
  })

  // A troca acontece uma vez, na transição. Depois dela o mestre manda: ir ao
  // bestiário com o tabuleiro aberto não pode ser desfeito pela própria cena.
  it('não rouba a aba de volta depois que o mestre escolhe outra', async () => {
    const rt = new FakeRealtime()
    const user = renderCena(rt)
    await screen.findByRole('tab', { name: /Tabuleiro/ })
    rt.abrirTabuleiro()
    await screen.findByText('Cripta')

    await user.click(screen.getByRole('tab', { name: /Bestiário/ }))

    expect(screen.queryByText('Cripta')).not.toBeInTheDocument()
  })
})
