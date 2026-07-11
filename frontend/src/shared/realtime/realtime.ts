import { useEffect, useMemo, useRef, useState } from 'react'
import { io, type Socket } from 'socket.io-client'

/**
 * Session runtime state, mirroring the backend `SessionRuntimeState`.
 * The shapes are duplicated because t20-data doesn't own the realtime
 * schema and cross-package sharing would drag Nest deps into the frontend
 * bundle. If drift becomes a real issue, promote both sides to
 * `@tormenta20/realtime-types`.
 */
export type InitiativeEntry = {
  id: string
  label: string
  initiative: number
  type: 'character' | 'npc'
  characterId?: number
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

/** A participant currently connected to the session room. Mirrors the
 * backend `presence` broadcast (deduped by userId). */
export type PresenceUser = {
  userId: number
  name: string
  role: 'gm' | 'player'
}

const EMPTY_STATE: SessionRuntimeState = {
  initiative: [],
  round: 0,
  turnIndex: -1,
}

const REALTIME_ORIGIN = window.location.origin

/**
 * Open a socket to the backend realtime gateway. Auth comes from the
 * existing session cookie — the WS handshake picks it up automatically
 * when `withCredentials: true` is set. The token override is available
 * for tests or headless clients.
 */
export function connectSession(token?: string): Socket {
  return io(REALTIME_ORIGIN, {
    withCredentials: true,
    transports: ['websocket', 'polling'],
    auth: token ? { token } : undefined,
    autoConnect: false,
  })
}

type ScopedBody = { campaignId: number; sessionId: number }

/**
 * React hook — connects, joins the session room, subscribes to
 * `session-state` broadcasts and exposes mutation helpers. The socket
 * closes on unmount.
 */
export function useSessionSocket(campaignId: number, sessionId: number) {
  const [state, setState] = useState<SessionRuntimeState>(EMPTY_STATE)
  const [isConnected, setIsConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  /* `hasPersistenceWarning` reflects the backend's dirty flag. When
   * true, the server failed to persist the last mutation; UI should
   * render a "unsaved changes" hint. The flag flips back off on the
   * next successful retry. */
  const [hasPersistenceWarning, setHasPersistenceWarning] = useState(false)
  const [present, setPresent] = useState<PresenceUser[]>([])
  /* Transient banner after a GM rest — auto-clears so it reads as a
   * notification, not persistent state. */
  const [restFlash, setRestFlash] = useState<'scene' | 'day' | null>(null)
  const socketRef = useRef<Socket | null>(null)

  useEffect(() => {
    if (!restFlash) return
    const t = setTimeout(() => setRestFlash(null), 4000)
    return () => clearTimeout(t)
  }, [restFlash])

  useEffect(() => {
    const socket = connectSession()
    socketRef.current = socket

    const body: ScopedBody = { campaignId, sessionId }

    socket.on('connect', () => {
      setIsConnected(true)
      setError(null)
      socket.emit('join-session', body, (ack: unknown) => {
        if (typeof ack === 'object' && ack && 'joined' in ack) {
          socket.emit(
            'get-session-state',
            body,
            (state: SessionRuntimeState) => {
              setState(state)
            },
          )
        }
      })
    })
    socket.on('disconnect', () => {
      setIsConnected(false)
      setPresent([])
    })
    socket.on('unauthorized', (payload: { message?: string }) => {
      setError(payload?.message ?? 'Unauthorized')
    })
    socket.on('session-state', (next: SessionRuntimeState) => {
      setState(next)
    })
    socket.on(
      'persistence-warning',
      (payload: { sessionId: number; dirty: boolean }) => {
        setHasPersistenceWarning(Boolean(payload?.dirty))
      },
    )
    socket.on('presence', (payload: { users?: PresenceUser[] }) => {
      setPresent(payload?.users ?? [])
    })
    socket.on('session-rest', (payload: { scope?: 'scene' | 'day' }) => {
      if (payload?.scope) setRestFlash(payload.scope)
    })

    socket.connect()

    return () => {
      socket.emit('leave-session', { sessionId })
      socket.disconnect()
      socketRef.current = null
    }
  }, [campaignId, sessionId])

  const actions = useMemo(
    () => ({
      addEntry: (entry: Omit<InitiativeEntry, 'id'>) => {
        socketRef.current?.emit('initiative-add', {
          campaignId,
          sessionId,
          entry,
        })
      },
      updateEntry: (entryId: string, patch: Partial<InitiativeEntry>) => {
        socketRef.current?.emit('initiative-update', {
          campaignId,
          sessionId,
          entryId,
          patch,
        })
      },
      removeEntry: (entryId: string) => {
        socketRef.current?.emit('initiative-remove', {
          campaignId,
          sessionId,
          entryId,
        })
      },
      nextTurn: () => {
        socketRef.current?.emit('initiative-next-turn', {
          campaignId,
          sessionId,
        })
      },
      resetInitiative: () => {
        socketRef.current?.emit('initiative-reset', { campaignId, sessionId })
      },
      populateParty: () => {
        socketRef.current?.emit('initiative-populate', {
          campaignId,
          sessionId,
        })
      },
      rest: (
        scope: 'scene' | 'day',
        condition?: 'ruim' | 'normal' | 'confortavel' | 'luxuosa',
      ) => {
        socketRef.current?.emit('session-rest', {
          campaignId,
          sessionId,
          scope,
          condition,
        })
      },
      patchVitals: (
        entryId: string,
        patch: { hpCurrent?: number; mpCurrent?: number },
      ) => {
        socketRef.current?.emit('vitals-patch', {
          campaignId,
          sessionId,
          entryId,
          patch,
        })
      },
      deltaVitals: (
        entryId: string,
        delta: { hpDelta?: number; mpDelta?: number },
      ) => {
        socketRef.current?.emit('vitals-delta', {
          campaignId,
          sessionId,
          entryId,
          ...delta,
        })
      },
    }),
    [campaignId, sessionId],
  )

  return {
    state,
    isConnected,
    error,
    hasPersistenceWarning,
    present,
    restFlash,
    ...actions,
  }
}
