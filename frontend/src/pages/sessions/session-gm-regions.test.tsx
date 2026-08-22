import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen, waitFor, within } from '@solidjs/testing-library'
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
import type { BoardState, InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { SessionGmView } from './session-gm-view'

/**
 * A geometria da cena do mestre (ALE-198).
 *
 * O que se prova aqui é o que o mestre notaria: **o tabuleiro nunca sai da
 * tela**, e tudo o mais é consulta que abre e fecha por cima dele. Antes desta
 * issue a região permanente trocava de conteúdo por aba e de LARGURA por
 * estado — quatro proporções condicionais, uma delas movendo a tela 255px em
 * cima do clique do mestre.
 *
 * As asserções são de CONTEÚDO e não de pixel: em jsdom todo elemento mede
 * zero, e afirmar largura aqui passaria verde sobre uma cena quebrada. O que
 * prova a estabilidade é o mapa continuar montado enquanto se abre cada coisa
 * — a largura fica para o e2e e para a validação nos seis formatos.
 */
const ARCANISTA: InitiativeEntry = {
  id: 'e1',
  label: 'Arcanista Erudito',
  initiative: 18,
  type: 'npc',
  hpCurrent: 42,
  hpMax: 42,
}

class FakeRealtime {
  private readonly live = createSignal<BoardState | null>(null)
  readonly updateEntry = vi.fn()
  readonly applyEffect = vi.fn()

  constructor(private readonly initiative: readonly InitiativeEntry[] = [ARCANISTA]) {}

  /** Abre o tabuleiro DEPOIS da montagem, que é como ele chega: pelo socket. */
  abrirTabuleiro() {
    this.live[1]({ version: 1, place: 'Cripta', terrain: 'pedra', tokens: [] })
  }

  asRealtime(): SessionRealtime {
    return {
      state: () => ({ initiative: this.initiative, round: 1, turnIndex: 0 }),
      isConnected: () => true,
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      restFlash: () => null,
      board: this.live[0],
      // O acervo de Lugares é do mestre e chega por PERGUNTA, não pelo snapshot
      // (ALE-124, fatia 5): o fake responde vazio.
      listPlaces: () => Promise.resolve([]),
      rest: vi.fn(),
      nextTurn: vi.fn(),
      previousTurn: vi.fn(),
      resetInitiative: vi.fn(),
      populateParty: vi.fn(),
      openBoard: vi.fn(),
      addEntry: vi.fn(),
      removeEntry: vi.fn(),
      updateEntry: this.updateEntry,
      applyEffect: this.applyEffect,
    } as unknown as SessionRealtime
  }
}

const SESSAO = { id: 4, campaignId: 1, sessionNumber: 4, status: 'active' } as unknown as Session

/** Simula uma faixa de largura. A cena tem UM degrau agora — 1024, onde os
 *  trilhos cabem ao lado do mapa —, contra os dois de antes. */
function comTrilhos(cabem: boolean) {
  window.matchMedia = vi.fn().mockImplementation((media: string) => ({
    // 1280 é onde a gaveta deixa de ser modal e passa a dividir a tela
    // (`SidePanel`): sem responder por ela, o teste ficaria preso na faixa em
    // que tudo atrás do overlay sai da árvore acessível.
    matches: media.includes('1024') || media.includes('1280') ? cabem : false,
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
        campaignName="Snapshot Test ALE-33"
        />
      </QueryClientProvider>
    ),
  })
  const router = createRouter({ routeTree: root, history: createMemoryHistory() })
  render(() => <RouterProvider router={router} />)
  return userEvent.setup()
}

/** O mapa está montado. Sem tabuleiro aberto é o convite que o representa —
 *  ele ocupa a MESMA região, e é isso que o torna a superfície permanente. */
const mapaNaTela = () => screen.getByText(/Nenhum tabuleiro aberto/)

beforeEach(() => comTrilhos(true))

describe('a superfície permanente da cena do mestre', () => {
  // `findBy` na primeira busca: o RouterProvider monta a rota num microtask, e
  // uma busca síncrona olharia a árvore antes de a cena existir.
  it('o tabuleiro está na tela sem ninguém pedir por ele', async () => {
    renderCena()

    expect(await screen.findByText(/Nenhum tabuleiro aberto/)).toBeInTheDocument()
  })

  // A queixa que abriu a issue: as abas tinham todas o mesmo tamanho, menos a de
  // combatente sem ninguém selecionado, e a tela pulava. A resposta não foi
  // igualar as larguras — foi tirar a região que trocava de conteúdo.
  it('não há mais abas trocando a região, nem seletor de região', async () => {
    renderCena()

    await screen.findByText(/Nenhum tabuleiro aberto/)
    expect(screen.queryByRole('tab', { name: /Combatente/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: /Tabuleiro/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'combate' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'mesa' })).not.toBeInTheDocument()
  })

  it('a fila do combate vive num trilho, com quem está na vez marcado', async () => {
    renderCena()

    const trilho = await screen.findByRole('navigation', { name: 'Fila do combate' })
    const item = within(trilho).getByRole('button', { name: /Arcanista Erudito/ })

    // O nome inteiro e os PV vão no nome acessível: no trilho o que se vê são
    // duas letras, e duas letras não são um nome.
    expect(item).toHaveAccessibleName('Abrir Arcanista Erudito — PV 42 de 42 — na vez')
  })
})

describe('a ficha do combatente como overlay', () => {
  it('clicar no trilho abre a ficha SEM tirar o mapa da tela', async () => {
    const user = renderCena()

    const trilho = await screen.findByRole('navigation', { name: 'Fila do combate' })
    await user.click(within(trilho).getByRole('button', { name: /Arcanista Erudito/ }))

    expect(await screen.findByRole('dialog', { name: 'Ficha de Arcanista Erudito' })).toBeInTheDocument()
    // O mapa continua montado por baixo: é ele que fica, e a ficha é a visita.
    // A consulta é por TEXTO e não por papel de propósito — o diálogo modal
    // marca tudo atrás como `aria-hidden`, e uma busca por papel devolveria
    // zero sobre uma cena inteiramente correta (gotcha do guia do front).
    expect(mapaNaTela()).toBeInTheDocument()
  })

  it('fechar a ficha devolve a cena sem ela', async () => {
    const user = renderCena()

    const trilho = await screen.findByRole('navigation', { name: 'Fila do combate' })
    await user.click(within(trilho).getByRole('button', { name: /Arcanista Erudito/ }))
    const ficha = await screen.findByRole('dialog', { name: 'Ficha de Arcanista Erudito' })

    await user.click(within(ficha).getByRole('button', { name: 'Fechar o combatente' }))

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Ficha de Arcanista Erudito' })).not.toBeInTheDocument(),
    )
    expect(mapaNaTela()).toBeInTheDocument()
  })
})

describe('as consultas do mestre', () => {
  it('escolher alguém na gaveta da fila FECHA a gaveta', async () => {
    const user = renderCena()

    const trilho = await screen.findByRole('navigation', { name: 'Fila do combate' })
    await user.click(within(trilho).getByRole('button', { name: 'Abrir a iniciativa' }))
    const gaveta = await screen.findByRole('dialog', { name: 'Iniciativa' })

    await user.click(within(gaveta).getByRole('button', { name: 'Arcanista Erudito' }))

    // O gesto termina onde começou: abriu-se a fila para achar alguém, achou-se.
    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Iniciativa' })).not.toBeInTheDocument(),
    )
  })

  // Um overlay por vez, nunca empilhados — é o que separa isto dos side sheets
  // que a ALE-122 matou, onde um painel abria POR CIMA do outro.
  it('abrir uma consulta fecha a gaveta da fila', async () => {
    const user = renderCena()

    const trilho = await screen.findByRole('navigation', { name: 'Fila do combate' })
    await user.click(within(trilho).getByRole('button', { name: 'Abrir a iniciativa' }))
    await screen.findByRole('dialog', { name: 'Iniciativa' })

    const consultas = screen.getByRole('navigation', { name: 'Consultas do mestre' })
    await user.click(within(consultas).getByRole('button', { name: 'Catálogos' }))

    await waitFor(() =>
      expect(screen.queryByRole('dialog', { name: 'Iniciativa' })).not.toBeInTheDocument(),
    )
    expect(await screen.findByRole('dialog', { name: 'Catálogos' })).toBeInTheDocument()
  })

  // As notas são a exceção declarada ao overlay: elas EMPURRAM o mapa em vez de
  // cobri-lo, porque se escrevem enquanto se narra olhando o tabuleiro.
  it('as notas abrem ao lado do mapa, não por cima dele', async () => {
    const user = renderCena()

    await screen.findByText(/Nenhum tabuleiro aberto/)
    const consultas = screen.getByRole('navigation', { name: 'Consultas do mestre' })
    await user.click(within(consultas).getByRole('button', { name: 'Notas' }))

    // Os dois na tela ao mesmo tempo, e a nota NÃO é diálogo: fosse overlay,
    // o mapa teria saído da árvore acessível junto.
    expect(await screen.findByRole('textbox', { name: /Notas/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Notas' })).toHaveAttribute('aria-pressed', 'true')
    expect(mapaNaTela()).toBeInTheDocument()
  })
})

describe('abaixo do degrau dos trilhos', () => {
  beforeEach(() => comTrilhos(false))

  // A fila do combate não cabe como trilho num telefone: 64px são 16% da tela.
  // Ela some, e o MESMO botão a alcança — um caminho só, em toda largura.
  it('a fila sai da tela e o botão da fileira abre a mesma gaveta', async () => {
    const user = renderCena()

    await screen.findByText(/Nenhum tabuleiro aberto/)
    expect(screen.queryByRole('button', { name: 'Abrir a iniciativa' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Iniciativa · 1/ }))

    expect(await screen.findByRole('dialog', { name: 'Iniciativa' })).toBeInTheDocument()
  })
})
