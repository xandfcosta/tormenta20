import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { render, screen } from '@solidjs/testing-library'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Session } from '@/shared/api/api'
import type {
  InitiativeEntry,
  SessionRealtime,
  SessionRuntimeState,
} from '@/shared/realtime/realtime'
import { SessionGmView } from './session-gm-view'

/**
 * A faixa do turno — o que o mestre mais clica na sessão.
 *
 * "Turno anterior" existe porque um "Próximo turno" a mais é o erro mais comum
 * da mesa, e o conserto até aqui era dar a volta na iniciativa inteira, o que
 * empurrava a RODADA junto (ALE-122).
 *
 * Desde a ALE-210 a vaga do avanço é a vaga do CICLO da cena, e é por isso que
 * o ciclo se testa aqui: a regra que ele carrega não é "existe um botão de
 * iniciar", é que a vaga contém EXATAMENTE UM "começar" por vez.
 */
class FakeRealtime {
  readonly nextTurn = vi.fn()
  readonly previousTurn = vi.fn()
  readonly resetInitiative = vi.fn()
  readonly startScene = vi.fn()
  readonly endScene = vi.fn()

  constructor(private readonly state: SessionRuntimeState) {}

  asRealtime(): SessionRealtime {
    return {
      state: () => this.state,
      isConnected: () => true,
      // A cena do mestre passou a montar a região do tabuleiro (ALE-124): sem
      // este acessor o fake mente sobre a forma do contrato e a tela quebra.
      board: () => null,
      // O acervo de Lugares é do mestre e chega por PERGUNTA, não pelo snapshot
      // (ALE-124, fatia 5): o fake responde vazio.
      listPlaces: () => Promise.resolve([]),
      error: () => null,
      hasPersistenceWarning: () => false,
      present: () => [],
      nextTurn: this.nextTurn,
      previousTurn: this.previousTurn,
      resetInitiative: this.resetInitiative,
      startScene: this.startScene,
      endScene: this.endScene,
      addEntry: vi.fn(),
      updateEntry: vi.fn(),
      removeEntry: vi.fn(),
      populateParty: vi.fn(),
      deltaVitals: vi.fn(),
      applyEffect: vi.fn(),
      rest: vi.fn(),
    } as unknown as SessionRealtime
  }
}

const SESSION = { id: 4, campaignId: 1, sessionNumber: 4, title: null } as unknown as Session

const linha = (id: string, label: string): InitiativeEntry =>
  ({ id, label, initiative: 10, type: 'npc' }) as InitiativeEntry

/** Cena rolando: dois combatentes e o primeiro na vez. */
const EM_COMBATE: SessionRuntimeState = {
  initiative: [linha('a', 'Arcanista'), linha('b', 'Ogro')],
  round: 1,
  turnIndex: 0,
  turnsTaken: 1,
  sceneActive: true,
}

/** Sessão recém-aberta: sem cena e sem fila. */
const FORA_DE_CENA: SessionRuntimeState = {
  initiative: [],
  round: 0,
  turnIndex: -1,
  turnsTaken: 0,
  sceneActive: false,
}

function renderScene(state: SessionRuntimeState = EM_COMBATE) {
  const rt = new FakeRealtime(state)
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(() => (
    <QueryClientProvider client={client}>
      <SessionGmView
        campaignId={1}
        sessionId={4}
        session={SESSION}
        rt={rt.asRealtime()}
        myCharacterIds={new Set<number>()}
        campaignName="Snapshot Test ALE-33"
        members={[]}
      />
    </QueryClientProvider>
  ))
  return { rt, user: userEvent.setup() }
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
})

describe('faixa do turno', () => {
  it('desfazer um turno pede o turno anterior ao servidor', async () => {
    const { rt, user } = renderScene()

    await user.click(screen.getByRole('button', { name: 'Turno anterior' }))

    expect(rt.previousTurn).toHaveBeenCalledTimes(1)
    expect(rt.nextTurn).not.toHaveBeenCalled()
  })

  // Reiniciar apaga o combate inteiro e fica longe do avanço, atrás de
  // confirmação — a proteção estava no lugar errado antes (ALE-122). Na
  // ALE-184 ele saiu da FAIXA para o menu da sessão, e na ALE-210 saiu do menu
  // para o pé do trilho, junto do resto do ciclo da cena: continua longe do
  // botão mais clicado e continua atrás de confirmação.
  it('reiniciar fica fora da faixa e continua pedindo confirmação', async () => {
    const { rt, user } = renderScene()
    const faixa = screen.getByText('Rodada 1 · Turno 1/2').closest('div')

    const reiniciar = screen.getByRole('button', { name: 'Reiniciar o combate' })
    expect(faixa).not.toContainElement(reiniciar)

    await user.click(reiniciar)

    expect(rt.resetInitiative).not.toHaveBeenCalled()
    expect(await screen.findByRole('dialog')).toHaveTextContent('Reiniciar o combate?')
  })

  // O avanço passa a dizer PARA ONDE vai (ALE-184). Com a lista vazia não há
  // destino, e desde a ALE-210 esse estado só existe DENTRO da cena — o mestre
  // acabou de iniciar e ainda vai montar a ordem —, então o rótulo diz o que
  // falta em vez do verbo que não vai acontecer.
  it('em cena sem ninguém na fila, o avanço diz o que falta', () => {
    renderScene({ ...FORA_DE_CENA, sceneActive: true })

    expect(screen.getAllByRole('button', { name: 'Ninguém na fila' })[0]).toBeDisabled()
  })
})

/**
 * O CICLO DA CENA (ALE-210).
 *
 * A cena virou estado que o mestre liga e desliga, e a trava que esconde a fila
 * dos jogadores é do SERVIDOR — ela se prova em `session_state_test.go`, não
 * aqui. O que só esta camada pode provar é a composição: que a vaga do avanço
 * contém um "começar" por vez, e que o fim do ciclo continua alcançável quando
 * a cena acaba e a fila fica.
 */
describe('o ciclo da cena', () => {
  it('fora de cena a faixa oferece INICIAR, e nenhum outro começar existe', async () => {
    const { rt, user } = renderScene(FORA_DE_CENA)

    // A segunda metade é a que importa: perguntar só pelo "Iniciar cena"
    // passaria verde com os dois botões na tela ao mesmo tempo, que é
    // exatamente o risco que a vaga única existe para eliminar.
    expect(screen.queryByRole('button', { name: /^Começar/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Ninguém na fila' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Turno anterior' })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Iniciar cena' }))

    expect(rt.startScene).toHaveBeenCalledTimes(1)
  })

  it('em cena a vaga volta a ser o avanço, e o iniciar some', () => {
    renderScene()

    expect(screen.queryByRole('button', { name: 'Iniciar cena' })).not.toBeInTheDocument()
    expect(screen.getAllByRole('button', { name: 'Próximo: Ogro' })[0]).toBeEnabled()
  })

  it('encerrar a cena pergunta antes, e diz que a fila sobrevive', async () => {
    const { rt, user } = renderScene()

    await user.click(screen.getByRole('button', { name: 'Encerrar cena' }))

    expect(rt.endScene).not.toHaveBeenCalled()
    // Encerrar e reiniciar parecem a mesma coisa; o que os separa é o que
    // sobrevive, e por isso a pergunta tem de dizer isso na cara.
    expect(await screen.findByRole('dialog')).toHaveTextContent('CONTINUAM na lista')
  })

  // O buraco que a especificação abria: encerrar GUARDA os combatentes, então
  // depois dele existe "fora de cena com fila cheia". Amarrar o reiniciar à
  // cena deixaria essa lista sem nenhum caminho de saída — o mestre teria de
  // iniciar uma cena que não vai jogar só para poder esvaziá-la.
  it('encerrada a cena, a fila guardada ainda pode ser esvaziada', () => {
    renderScene({ ...EM_COMBATE, sceneActive: false, round: 0, turnIndex: -1 })

    expect(screen.getByRole('button', { name: 'Iniciar cena' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reiniciar o combate' })).toBeInTheDocument()
    // Encerrar, esse sim, não existe fora de cena: não há o que encerrar.
    expect(screen.queryByRole('button', { name: 'Encerrar cena' })).not.toBeInTheDocument()
  })
})
