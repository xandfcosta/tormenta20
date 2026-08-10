import type { SessionStatus } from '@/shared/api/api'

/** How a section colors a session's status: a live session pulses green
 *  (--hp-full, matching the tome header above it), a foretold one is gilt, a
 *  closed one recedes to iron. */
export type SessionTone = 'live' | 'planned' | 'ended'

export type SessionStatusMeta = { label: string; tone: SessionTone }

/**
 * Presentation meta for a session's status — its pt-BR label + the tone the
 * grimório sections map to color. Centralized so the overview, the log and any
 * badge read one source instead of re-deriving the ternary each place (it drifted
 * across three files before).
 *
 * @example sessionStatusMeta('active') // { label: 'Ao vivo', tone: 'live' }
 */
export function sessionStatusMeta(status: SessionStatus): SessionStatusMeta {
  if (status === 'active') return { label: 'Ao vivo', tone: 'live' }
  if (status === 'ended') return { label: 'Encerrada', tone: 'ended' }
  return { label: 'Planejada', tone: 'planned' }
}
