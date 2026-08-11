import { createRoot } from 'solid-js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  type SessionSocket,
  type SessionRuntimeState,
  createSessionSocket,
} from './realtime'

/**
 * Stands in for socket.io: records what was emitted and lets the test push
 * server events back. A named fake, not an inline stub — the realtime contract
 * is the thing under test, and it deserves a readable double.
 */
class FakeSocket implements SessionSocket {
  readonly emitted: { event: string; args: unknown[] }[] = []
  connected = false
  disconnectCount = 0
  private handlers = new Map<string, (payload: never) => void>()
  /** Acks the next `join-session` / `get-session-state` when set. */
  ackWith: Record<string, unknown> = {}

  on(event: string, handler: (payload: never) => void): void {
    this.handlers.set(event, handler)
  }

  emit(event: string, ...args: unknown[]): void {
    this.emitted.push({ event, args })
    const ack = args.find((arg): arg is (value: unknown) => void => typeof arg === 'function')
    if (ack && event in this.ackWith) ack(this.ackWith[event])
  }

  connect(): void {
    this.connected = true
  }

  disconnect(): void {
    this.connected = false
    this.disconnectCount++
  }

  /** Pretends the server sent an event. */
  server<T>(event: string, payload?: T): void {
    this.handlers.get(event)?.(payload as never)
  }

  emitsOf(event: string): unknown[][] {
    return this.emitted.filter((e) => e.event === event).map((e) => e.args)
  }
}

const STATE: SessionRuntimeState = {
  initiative: [{ id: 'a', label: 'Goblin', initiative: 14, type: 'npc' }],
  round: 2,
  turnIndex: 0,
}

function withSocket(run: (socket: FakeSocket, rt: ReturnType<typeof createSessionSocket>) => void) {
  const socket = new FakeSocket()
  createRoot((dispose) => {
    const rt = createSessionSocket(
      () => 1,
      () => 7,
      { connect: () => socket },
    )
    run(socket, rt)
    dispose()
  })
  return socket
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createSessionSocket — ciclo de vida', () => {
  it('conecta e entra na sala da sessão', () => {
    const socket = withSocket((socket) => {
      socket.ackWith = { 'join-session': { joined: 'session:7' } }
      socket.server('connect')

      expect(socket.connected).toBe(true)
      expect(socket.emitsOf('join-session')[0][0]).toEqual({ campaignId: 1, sessionId: 7 })
    })
    expect(socket.disconnectCount).toBe(1)
  })

  // O ack do join é o que libera pedir o estado — pedir antes traria a sala vazia.
  it('só pede o estado depois do ack do join', () => {
    withSocket((socket) => {
      socket.server('connect')
      expect(socket.emitsOf('get-session-state')).toHaveLength(0)

      socket.ackWith = { 'join-session': { joined: 'session:7' } }
      socket.server('connect')

      expect(socket.emitsOf('get-session-state')).toHaveLength(1)
    })
  })

  it('sai da sala antes de desconectar', () => {
    const socket = withSocket(() => {})

    const events = socket.emitted.map((e) => e.event)
    expect(events).toContain('leave-session')
    expect(events.indexOf('leave-session')).toBeLessThan(events.length)
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
  it('toda ação carrega o escopo da sessão', () => {
    withSocket((socket, rt) => {
      rt.nextTurn()
      rt.removeEntry('a')

      expect(socket.emitsOf('initiative-next-turn')[0][0]).toEqual({
        campaignId: 1,
        sessionId: 7,
      })
      expect(socket.emitsOf('initiative-remove')[0][0]).toMatchObject({ entryId: 'a' })
    })
  })

  it('o delta de vitais vai achatado, como o servidor lê', () => {
    withSocket((socket, rt) => {
      rt.deltaVitals('a', { hpDelta: -3 })

      expect(socket.emitsOf('vitals-delta')[0][0]).toEqual({
        campaignId: 1,
        sessionId: 7,
        entryId: 'a',
        hpDelta: -3,
      })
    })
  })

  it('iniciativa própria manda o personagem e o valor rolado', () => {
    withSocket((socket, rt) => {
      rt.rollSelfInitiative(42, 17)

      expect(socket.emitsOf('initiative-self')[0][0]).toMatchObject({
        characterId: 42,
        initiative: 17,
      })
    })
  })
})
