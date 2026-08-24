import { createRoot } from 'solid-js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  type BoardState,
  type SessionStream,
  type SessionRuntimeState,
  createSessionSocket,
} from './realtime'

/**
 * Stands in for socket.io: records what was emitted and lets the test push
 * server events back. A named fake, not an inline stub — the realtime contract
 * is the thing under test, and it deserves a readable double.
 */
/**
 * Duas metades, porque o transporte agora tem duas (ALE-253): o FLUXO desce
 * eventos e o COMANDO sobe. O socket fazia as duas com o mesmo objeto.
 */
class FakeStream implements SessionStream {
  closeCount = 0
  private handlers = new Map<string, (payload: never) => void>()

  on(event: string, handler: (payload: never) => void): void {
    this.handlers.set(event, handler)
  }

  close(): void {
    this.closeCount++
  }

  /** Finge que o servidor empurrou um evento. */
  server<T>(event: string, payload?: T): void {
    this.handlers.get(event)?.(payload as never)
  }
}

/** Grava o que SOBE e responde o que o teste mandar. */
class FakeCommands {
  readonly sent: { event: string; body: Record<string, unknown> }[] = []
  respondWith: Record<string, unknown> = {}

  readonly run = (
    event: string,
    _ids: { campaignId: number; sessionId: number },
    body: Record<string, unknown>,
  ): Promise<unknown> => {
    this.sent.push({ event, body })
    return Promise.resolve(this.respondWith[event] ?? null)
  }

  sentOf(event: string): Record<string, unknown>[] {
    return this.sent.filter((c) => c.event === event).map((c) => c.body)
  }
}

const STATE: SessionRuntimeState = {
  initiative: [{ id: 'a', label: 'Goblin', initiative: 14, type: 'npc' }],
  round: 2,
  turnIndex: 0,
  sceneActive: true,
}

/** Como o `withSocket`, mas com o aviso de ficha mudada ligado (ALE-245). */
function withCharacterWatch(run: (stream: FakeStream, avisados: number[]) => void) {
  const stream = new FakeStream()
  const comandos = new FakeCommands()
  const avisados: number[] = []
  createRoot((dispose) => {
    createSessionSocket(
      () => 1,
      () => 7,
      {
        connect: () => stream,
        command: comandos.run,
        onCharacterChanged: (id) => avisados.push(id),
      },
    )
    run(stream, avisados)
    dispose()
  })
  return avisados
}

function withSocket(
  run: (stream: FakeStream, rt: ReturnType<typeof createSessionSocket>, comandos: FakeCommands) => void,
) {
  const stream = new FakeStream()
  const comandos = new FakeCommands()
  createRoot((dispose) => {
    const rt = createSessionSocket(
      () => 1,
      () => 7,
      { connect: () => stream, command: comandos.run },
    )
    run(stream, rt, comandos)
    dispose()
  })
  return { stream, comandos }
}

/** Os corpos enviados de um comando. O ESCOPO não vem mais no corpo: ele está
 *  no caminho da rota, e é o `realtime-wire.test.ts` que o prende. */
const comandosDe = (c: FakeCommands, event: string) => c.sentOf(event)

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createSessionSocket — ciclo de vida', () => {
  // Sem `join-session` e sem ack: entrar na sala É abrir o fluxo, e o servidor
  // resolve a mesa pelo CAMINHO da requisição (ALE-253). O que sobreviveu da
  // regra antiga é o que importava — ao conectar, a tela se hidrata.
  it('conectar hidrata a tela por HTTP', () => {
    withSocket((stream, _rt, comandos) => {
      stream.server('connect')

      expect(comandosDe(comandos, 'get-session-state')).toHaveLength(1)
      expect(comandosDe(comandos, 'get-board-state')).toHaveLength(1)
    })
  })

  // SAIR é FECHAR o fluxo, e some a corrida que a regra antiga prendia: não há
  // mais um "leave" que possa sair DEPOIS do disconnect, porque o servidor tira
  // a presença pelo cancelamento do contexto da requisição.
  it('desmontar fecha o fluxo', () => {
    const { stream } = withSocket(() => {})

    expect(stream.closeCount).toBe(1)
  })
})

describe('createSessionSocket — estado', () => {
  it('o broadcast do servidor vira estado', () => {
    withSocket((socket, rt) => {
      socket.server('session-state', STATE)

      expect(rt.state().round).toBe(2)
      expect(rt.state().initiative[0].label).toBe('Goblin')
    })
  })

  it('conexão e queda mexem no indicador', () => {
    withSocket((socket, rt) => {
      expect(rt.isConnected()).toBe(false)
      socket.server('connect')
      expect(rt.isConnected()).toBe(true)

      socket.server('disconnect')

      expect(rt.isConnected()).toBe(false)
    })
  })

  // Queda de rede não pode deixar o roster mostrando gente que não está mais lá.
  it('desconectar limpa a presença', () => {
    withSocket((socket, rt) => {
      socket.server('presence', { users: [{ userId: 1, name: 'Mestre', role: 'gm' }] })
      expect(rt.present()).toHaveLength(1)

      socket.server('disconnect')

      expect(rt.present()).toEqual([])
    })
  })

  it('aviso de persistência acompanha a flag do servidor', () => {
    withSocket((socket, rt) => {
      socket.server('persistence-warning', { sessionId: 7, dirty: true })
      expect(rt.hasPersistenceWarning()).toBe(true)

      socket.server('persistence-warning', { sessionId: 7, dirty: false })

      expect(rt.hasPersistenceWarning()).toBe(false)
    })
  })

  it('recusa do servidor vira erro legível', () => {
    withSocket((socket, rt) => {
      socket.server('unauthorized', { message: 'Sessão de outro mestre' })

      expect(rt.error()).toBe('Sessão de outro mestre')
    })
  })

  // O descanso é NOTIFICAÇÃO, não estado: some sozinho.
  it('o flash de descanso se apaga sozinho', () => {
    withSocket((socket, rt) => {
      socket.server('session-rest', { scope: 'day' })
      expect(rt.restFlash()).toBe('day')

      vi.advanceTimersByTime(4000)

      expect(rt.restFlash()).toBeNull()
    })
  })
})

describe('createSessionSocket — ações', () => {
  // O ESCOPO saiu do corpo: ele está no CAMINHO da rota agora, e quem o prende
  // é o `realtime-wire.test.ts`. Aqui prova-se que a ação sai, e com o quê.
  it('cada ação sai como um comando próprio', () => {
    withSocket((_stream, rt, comandos) => {
      rt.nextTurn()
      rt.removeEntry('a')

      expect(comandosDe(comandos, 'initiative-next-turn')).toHaveLength(1)
      expect(comandosDe(comandos, 'initiative-remove')[0]).toMatchObject({ entryId: 'a' })
    })
  })

  it('o delta de vitais vai achatado, como o servidor lê', () => {
    withSocket((_stream, rt, comandos) => {
      rt.deltaVitals('a', { hpDelta: -3 })

      expect(comandosDe(comandos, 'vitals-delta')[0]).toEqual({ entryId: 'a', hpDelta: -3 })
    })
  })

  // O D20, e nunca o total (ALE-213): quem soma o bônus da perícia é o Go.
  it('iniciativa própria manda o personagem e o d20', () => {
    withSocket((_stream, rt, comandos) => {
      rt.rollSelfInitiative(42, 17)

      expect(comandosDe(comandos, 'initiative-self')[0]).toMatchObject({
        characterId: 42,
        d20: 17,
      })
    })
  })
})


/**
 * O tabuleiro (ALE-124) chega por um evento PRÓPRIO. O que se prova aqui é o
 * contrato do fio: o broadcast vira sinal, e o atrasado é DESCARTADO.
 */
describe('o tabuleiro no fio', () => {
  const TABULEIRO: BoardState = {
    version: 4,
    place: 'Taverna do Javali',
    terrain: 'taverna',
    tokens: [{ id: 't1', label: 'Ogro', x: 3, y: 4, footprint: 2, kind: 'npc' }],
  }

  it('o broadcast do tabuleiro vira sinal', () => {
    withSocket((socket, rt) => {
      socket.server('board-state', TABULEIRO)

      expect(rt.board()?.place).toBe('Taverna do Javali')
      expect(rt.board()?.tokens).toHaveLength(1)
    })
  })

  // Encerrar chega como `null` — e `null` sempre vale, porque encerrar é a única
  // mudança que não carrega versão.
  it('encerrar apaga o tabuleiro da tela', () => {
    withSocket((socket, rt) => {
      socket.server('board-state', TABULEIRO)
      socket.server('board-state', null)

      expect(rt.board()).toBeNull()
    })
  })

  // O caso invisível: um broadcast que ficou no buffer antes da queda chega
  // DEPOIS da re-hidratação. Sem a guarda, a tela volta no tempo — e ninguém
  // percebe, porque um tabuleiro atrasado ainda parece um tabuleiro.
  it('um estado com versão MENOR que a atual é descartado', () => {
    withSocket((socket, rt) => {
      socket.server('board-state', TABULEIRO)
      socket.server('board-state', { ...TABULEIRO, version: 2, place: 'Cripta' })

      expect(rt.board()?.place).toBe('Taverna do Javali')
      expect(rt.board()?.version).toBe(4)
    })
  })

  it('entrar na sessão já pede o tabuleiro', () => {
    withSocket((stream, _rt, comandos) => {
      stream.server('connect')

      expect(comandosDe(comandos, 'get-board-state')).toHaveLength(1)
    })
  })

  it('mover a peça sai com o id e o quadrado', () => {
    withSocket((_stream, rt, comandos) => {
      rt.updateToken('t1', { x: 5, y: 6 })

      expect(comandosDe(comandos, 'board-token-update')[0]).toEqual({
        tokenId: 't1',
        patch: { x: 5, y: 6 },
      })
    })
  })
})

/**
 * A FICHA MUDOU NO SERVIDOR E QUEM ESTÁ OLHANDO PRECISA SABER (ALE-245).
 *
 * O mestre aplica "Caído" num PC pela ficha do combatente; a escrita é HTTP, e
 * até esta fatia nenhum handler HTTP conseguia falar com a sala — o gateway
 * guarda `s *Server` e o ponteiro nunca vai na direção contrária. A tela do
 * jogador ficava com a condição faltando E com Defesa e perícias derivadas do
 * estado velho (ALE-28), sem nada dizendo que os dois discordavam.
 *
 * O transporte não conhece o cache de propósito: ele só entrega o id, e quem
 * invalida é a página, que é quem tem o `queryClient`.
 */
describe('character-changed', () => {
  it('entrega o id de quem mudou', () => {
    const avisados = withCharacterWatch((stream) => {
      stream.server('connect')
      stream.server('character-changed', { characterId: 14 })
    })

    expect(avisados).toEqual([14])
  })

  // DUAS mudanças do MESMO personagem contam duas vezes, e é por isso que o
  // aviso é retorno de chamada e não sinal: um sinal guardando o último id não
  // notifica ao receber o mesmo valor, e aplicar duas condições em sequência no
  // mesmo PC é o caso comum.
  it('o mesmo personagem duas vezes avisa duas vezes', () => {
    const avisados = withCharacterWatch((stream) => {
      stream.server('connect')
      stream.server('character-changed', { characterId: 14 })
      stream.server('character-changed', { characterId: 14 })
    })

    expect(avisados).toEqual([14, 14])
  })

  // Corpo sem id não vira invalidação de `undefined`, que derrubaria o cache
  // inteiro de personagens por uma mensagem malformada.
  it('corpo sem id é ignorado', () => {
    const avisados = withCharacterWatch((stream) => {
      stream.server('connect')
      stream.server('character-changed', {})
      stream.server('character-changed', { characterId: 'catorze' })
    })

    expect(avisados).toEqual([])
  })
})
