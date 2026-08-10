import { Link } from '@tanstack/react-router'
import { ChevronRight } from 'lucide-react'
import type { Session } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import {
  type SessionStatusMeta,
  type SessionTone,
  sessionStatusMeta,
} from '@/entities/session/status'
import { orderSessionsForLog } from '@/entities/session/log-order'

// Tone tints the seal's border + number only — never its background. The base
// keeps an OPAQUE fill (grimório bg) so the rail line stays hidden behind every
// node; a translucent tint here would let the line show through (live-node bug).
const SEAL_TONE: Record<SessionTone, string> = {
  live: 'border-[color:var(--hp-full)] text-[color:var(--hp-full)]',
  planned: 'border-grimorio-gold/60 text-grimorio-gold',
  ended: 'border-grimorio-iron text-muted-foreground',
}

const PILL_TONE: Record<SessionTone, string> = {
  live: 'border-[color:var(--hp-full)]/70 text-[color:var(--hp-full)] bg-[color:var(--hp-full)]/12',
  planned: 'border-grimorio-gold/50 text-grimorio-gold bg-grimorio-gold/10',
  ended: 'border-grimorio-iron text-muted-foreground',
}

/**
 * The chronicle log: sessions as entries down a gilt rail, the live one first
 * and highlighted (green seal + pulse + "Entrar"). `limit` trims it to the most
 * relevant few for the Visão geral preview; omitted shows the full log.
 */
export function SessionLog({
  sessions,
  campaignId,
  limit,
}: {
  sessions: Session[]
  campaignId: number
  limit?: number
}) {
  const ordered = orderSessionsForLog(sessions)
  const shown = limit ? ordered.slice(0, limit) : ordered
  return (
    <ol className="relative space-y-3 py-1">
      <span
        aria-hidden
        className="absolute bottom-3 left-4 top-3 w-px bg-grimorio-iron/70"
      />
      {shown.map((session) => (
        <SessionLogEntry key={session.id} session={session} campaignId={campaignId} />
      ))}
    </ol>
  )
}

function SessionLogEntry({
  session,
  campaignId,
}: {
  session: Session
  campaignId: number
}) {
  const meta = sessionStatusMeta(session.status)
  const isLive = meta.tone === 'live'
  return (
    <li className="relative pl-11">
      <SessionSeal number={session.sessionNumber} tone={meta.tone} />
      <Link
        to="/campaigns/$id/sessions/$sid"
        params={{ id: campaignId, sid: session.id }}
        className={cn(
          'flex items-center justify-between gap-3 rounded-sm border bg-[var(--grimorio-panel)] p-3 transition-colors',
          isLive
            ? 'border-[color:var(--hp-full)]/60 bg-[color:var(--hp-full)]/[0.06] hover:border-[color:var(--hp-full)]'
            : 'border-grimorio-iron hover:border-grimorio-gold',
        )}
      >
        <div className="min-w-0">
          <p className="truncate font-medium text-foreground">
            Sessão {session.sessionNumber}
            {session.title && (
              <span className="text-muted-foreground"> — {session.title}</span>
            )}
          </p>
          <p className="text-xs text-muted-foreground">
            {new Date(session.createdAt).toLocaleDateString('pt-BR')}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <StatusPill meta={meta} />
          {isLive && (
            <span className="hidden items-center gap-1 rounded-sm bg-[color:var(--hp-full)]/15 px-2 py-1 text-xs font-semibold text-[color:var(--hp-full)] sm:inline-flex">
              Entrar
              <ChevronRight aria-hidden className="size-3.5" />
            </span>
          )}
        </div>
      </Link>
    </li>
  )
}

/** The rail node — the session's number as a wax-seal disc, tinted by tone. */
function SessionSeal({ number, tone }: { number: number; tone: SessionTone }) {
  return (
    <span
      aria-hidden
      className={cn(
        'absolute left-0 top-2.5 flex size-8 items-center justify-center rounded-full border bg-[var(--grimorio-bg)] font-heading text-sm',
        SEAL_TONE[tone],
      )}
    >
      {number}
    </span>
  )
}

/** Status chip: live pulses green, planned is gilt, ended recedes to iron. */
function StatusPill({ meta }: { meta: SessionStatusMeta }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-sm border px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider',
        PILL_TONE[meta.tone],
      )}
    >
      {meta.tone === 'live' && (
        <span className="size-1.5 animate-pulse rounded-full bg-[color:var(--hp-full)] motion-reduce:animate-none" />
      )}
      {meta.label}
    </span>
  )
}
