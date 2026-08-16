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
  hpCurrent?: number
  hpMax?: number
  mpCurrent?: number
  mpMax?: number
}

export type SessionRuntimeState = {
  initiative: InitiativeEntry[]
  round: number
  turnIndex: number
}

/** Someone connected to the session room (deduped by userId server-side). */
export type PresenceUser = {
  userId: number
  name: string
  role: 'gm' | 'player'
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

const EMPTY_STATE: SessionRuntimeState = { initiative: [], round: 0, turnIndex: -1 }

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
  }
}
