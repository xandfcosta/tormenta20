import { type Accessor, createRenderEffect, createSignal, onCleanup } from 'solid-js'
import { io } from 'socket.io-client'

/**
 * Session runtime state, mirroring the backend `SessionRuntimeState`. The
 * shapes are duplicated because t20-data doesn't own the realtime schema; if
 * drift ever bites, promote both sides to a shared package.
 */
export type InitiativeEntry = {
  id: string
  label: string
  initiative: number
  type: 'character' | 'npc'
  characterId?: number
  /** Verbete do bestiário de onde a linha veio — ausente em NPC digitado à mão. */
  monsterId?: string
  /**
   * Bloco de criatura que o MESTRE escreveu (ALE-137). Diferente do
   * `monsterId`, que aponta para o verbete imutável do livro: este é editável e
   * pertence à campanha. Uma linha tem um ou outro, nunca os dois.
   */
  creatureId?: number
  /** O mestre escondeu os PV desta linha: o jogador recebe a marca, não os números. */
  hpHidden?: boolean
  hpCurrent?: number
  hpMax?: number
  mpCurrent?: number
  mpMax?: number
}

export type SessionRuntimeState = {
  initiative: InitiativeEntry[]
  round: number
  turnIndex: number
  /**
   * Turnos desde o começo do combate. Vem CONTADO do servidor porque a conta
   * derivada (rodada × tamanho da lista) mente assim que alguém entra ou morre
   * no meio — que é o normal numa mesa (ALE-142). Opcional: uma sessão gravada
   * antes deste campo volta sem ele.
   */
  turnsTaken?: number
}

/** Someone connected to the session room (deduped by userId server-side). */
export type PresenceUser = {
  userId: number
  name: string
  role: 'gm' | 'player'
}

/**
 * Uma peça no tabuleiro (ALE-124). `x`/`y` são o canto superior-esquerdo em
 * QUADRADOS — nunca em pixels: pixel amarraria a posição ao tamanho da tela, e o
 * celular e o desktop passariam a discordar sobre onde o ogro está.
 */
export type BoardToken = {
  id: string
  label: string
  x: number
  y: number
  /** Lado da peça em quadrados (T20 p107): 1, 2, 3 ou 6. */
  footprint: number
  kind: 'character' | 'npc' | 'object'
  /** Linha da iniciativa correspondente — ausente em peça de cenário. */
  entryId?: string
  characterId?: number
  /** O mestre escondeu a peça; o jogador nem a recebe (some no servidor). */
  hidden?: boolean
}

/**
 * O tabuleiro da sessão. NÃO tem largura nem altura: o plano é infinito nas
 * quatro direções e a peça pode estar em coordenada negativa. Quem tem tamanho é
 * a JANELA, que mora no cliente — dois jogadores olhando pedaços diferentes da
 * mesma cena é propriedade, não defeito (ALE-124).
 */
export type BoardState = {
  /** Sobe a cada mutação aceita: é o que deixa o cliente descartar broadcast atrasado. */
  version: number
  place: string
  terrain: string
  tokens: BoardToken[]
}

export type RestScope = 'scene' | 'day'
export type RestCondition = 'ruim' | 'normal' | 'confortavel' | 'luxuosa'

/**
 * The slice of socket.io this app uses. Owning the interface is what lets the
 * tests drive a FakeSocket — and keeps `socket.io-client` from leaking past
 * this module (ALE-91 kept WS, but the seam is what makes revisiting cheap).
 */
export type SessionSocket = {
  on: (event: string, handler: (payload: never) => void) => void
  emit: (event: string, ...args: unknown[]) => void
  connect: () => void
  disconnect: () => void
}

const EMPTY_STATE: SessionRuntimeState = { initiative: [], round: 0, turnIndex: -1, turnsTaken: 0 }

/** How long the rest banner stays up before it reads as stale state. */
const REST_FLASH_MS = 4000

/**
 * Opens a socket to the realtime gateway. Auth rides the session cookie — the
 * WS handshake picks it up with `withCredentials`. The token override exists
 * for headless clients.
 */
export function connectSession(token?: string): SessionSocket {
  return io(window.location.origin, {
    withCredentials: true,
    transports: ['websocket', 'polling'],
    auth: token ? { token } : undefined,
    autoConnect: false,
  }) as unknown as SessionSocket
}

export type SessionRealtime = {
  state: Accessor<SessionRuntimeState>
  isConnected: Accessor<boolean>
  error: Accessor<string | null>
  /** The server failed to persist the last mutation; flips back on retry. */
  hasPersistenceWarning: Accessor<boolean>
  present: Accessor<PresenceUser[]>
  restFlash: Accessor<RestScope | null>
  addEntry: (entry: Omit<InitiativeEntry, 'id'>) => void
  updateEntry: (entryId: string, patch: Partial<InitiativeEntry>) => void
  removeEntry: (entryId: string) => void
  nextTurn: () => void
  /** Desfaz um turno — inclusive a virada de rodada. */
  previousTurn: () => void
  resetInitiative: () => void
  populateParty: () => void
  /** A player submits their own rolled initiative; upserts by characterId. */
  rollSelfInitiative: (characterId: number, initiative: number) => void
  rest: (scope: RestScope, condition?: RestCondition) => void
  patchVitals: (entryId: string, patch: { hpCurrent?: number; mpCurrent?: number }) => void
  deltaVitals: (entryId: string, delta: { hpDelta?: number; mpDelta?: number }) => void
  /** GM applies a spell buff to a combatant. Never automatic — the GM targets. */
  applyEffect: (entryId: string, spellId: string, scope?: RestScope) => void
  /** O tabuleiro da sessão — `null` quando o mestre não abriu nenhum. */
  board: Accessor<BoardState | null>
  openBoard: (place: string, terrain: string) => void
  closeBoard: () => void
  addToken: (token: Omit<BoardToken, 'id'>) => void
  removeToken: (tokenId: string) => void
  updateToken: (tokenId: string, patch: Partial<Omit<BoardToken, 'id'>>) => void
  /** Traz para o tabuleiro quem já está na iniciativa. Idempotente. */
  populateBoard: () => void
}

/**
 * Connects, joins the session room, turns broadcasts into signals and exposes
 * the mutations. `create*` because it OWNS the socket and the flash timer
 * across calls: it must be born once in a component body, never per event.
 *
 * Takes accessors so a route change to another session reconnects instead of
 * silently talking to the old room.
 *
 * @example const rt = createSessionSocket(() => campaignId(), () => sessionId())
 */
export function createSessionSocket(
  campaignId: Accessor<number>,
  sessionId: Accessor<number>,
  options: { connect?: (token?: string) => SessionSocket } = {},
): SessionRealtime {
  const [state, setState] = createSignal<SessionRuntimeState>(EMPTY_STATE)
  const [isConnected, setIsConnected] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  const [hasPersistenceWarning, setHasPersistenceWarning] = createSignal(false)
  const [present, setPresent] = createSignal<PresenceUser[]>([])
  const [restFlash, setRestFlash] = createSignal<RestScope | null>(null)
  const [board, setBoard] = createSignal<BoardState | null>(null)

  // O broadcast atrasado é real: um `board-state` que ficou no buffer antes da
  // queda chega DEPOIS da re-hidratação e faria a tela voltar no tempo. A versão
  // é monotônica no servidor, então comparar é o bastante — e `null` (tabuleiro
  // encerrado) sempre vale, porque encerrar é a única coisa que não tem versão.
  const acceptBoard = (next: BoardState | null) => {
    const current = board()
    if (next && current && next.version < current.version) return
    setBoard(next)
  }

  let socket: SessionSocket | null = null
  const open = options.connect ?? connectSession

  // The rest banner is a NOTIFICATION: it clears itself, or it reads as state.
  // The timer lives here rather than in an effect — an effect would tie a
  // notification to the render cycle, and a second rest would race the first.
  let flashTimer: ReturnType<typeof setTimeout> | undefined
  const flashRest = (scope: RestScope) => {
    clearTimeout(flashTimer)
    setRestFlash(scope)
    flashTimer = setTimeout(() => setRestFlash(null), REST_FLASH_MS)
  }
  onCleanup(() => clearTimeout(flashTimer))

  // createRenderEffect, not createEffect: the socket has to be up as soon as
  // the primitive is born — a deferred effect leaves the first actions of the
  // component firing into nothing.
  createRenderEffect(() => {
    const scope = { campaignId: campaignId(), sessionId: sessionId() }
    const live = open()
    socket = live

    live.on('connect', () => {
      setIsConnected(true)
      setError(null)
      // The state request waits for the join ack — asking earlier answers for
      // a room this socket has not entered yet.
      live.emit('join-session', scope, (ack: unknown) => {
        if (typeof ack !== 'object' || !ack || !('joined' in ack)) return
        live.emit('get-session-state', scope, (next: SessionRuntimeState) => setState(next))
        live.emit('get-board-state', scope, (next: BoardState | null) => setBoard(next ?? null))
      })
    })
    live.on('disconnect', () => {
      setIsConnected(false)
      // A dropped connection must not leave the roster showing people who left.
      setPresent([])
    })
    live.on('unauthorized', (payload: { message?: string }) =>
      setError(payload?.message ?? 'Sem permissão nesta sessão'),
    )
    live.on('session-state', (next: SessionRuntimeState) => setState(next))
    live.on('board-state', (next: BoardState | null) => acceptBoard(next ?? null))
    live.on('persistence-warning', (payload: { dirty?: boolean }) =>
      setHasPersistenceWarning(Boolean(payload?.dirty)),
    )
    live.on('presence', (payload: { users?: PresenceUser[] }) => setPresent(payload?.users ?? []))
    live.on('session-rest', (payload: { scope?: RestScope }) => {
      if (payload?.scope) flashRest(payload.scope)
    })

    live.connect()

    onCleanup(() => {
      live.emit('leave-session', { sessionId: scope.sessionId })
      live.disconnect()
      socket = null
    })
  })

  /** Every mutation carries the session scope; the server authorizes on it. */
  const send = (event: string, body: Record<string, unknown> = {}) => {
    socket?.emit(event, { campaignId: campaignId(), sessionId: sessionId(), ...body })
  }

  return {
    state,
    isConnected,
    error,
    hasPersistenceWarning,
    present,
    restFlash,

    addEntry: (entry) => send('initiative-add', { entry }),
    updateEntry: (entryId, patch) => send('initiative-update', { entryId, patch }),
    removeEntry: (entryId) => send('initiative-remove', { entryId }),
    nextTurn: () => send('initiative-next-turn'),
    previousTurn: () => send('initiative-previous-turn'),
    resetInitiative: () => send('initiative-reset'),
    populateParty: () => send('initiative-populate'),
    rollSelfInitiative: (characterId, initiative) =>
      send('initiative-self', { characterId, initiative }),
    rest: (scope, condition) => send('session-rest', { scope, condition }),
    patchVitals: (entryId, patch) => send('vitals-patch', { entryId, patch }),
    // Flat, not nested: this is the shape the server reads.
    deltaVitals: (entryId, delta) => send('vitals-delta', { entryId, ...delta }),
    applyEffect: (entryId, spellId, scope) =>
      send('apply-effect', { entryId, spellId, scope }),

    board,
    openBoard: (place, terrain) => send('board-open', { place, terrain }),
    closeBoard: () => send('board-close'),
    // Achatado, não aninhado: é a forma que o servidor lê.
    addToken: (token) => send('board-token-add', { ...token }),
    removeToken: (tokenId) => send('board-token-remove', { tokenId }),
    updateToken: (tokenId, patch) => send('board-token-update', { tokenId, patch }),
    populateBoard: () => send('board-populate'),
  }
}
