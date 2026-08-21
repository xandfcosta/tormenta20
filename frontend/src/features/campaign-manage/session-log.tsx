import { Link } from '@tanstack/solid-router'
import { ChevronRight } from 'lucide-solid'
import { For, Show } from 'solid-js'
import { orderSessionsForLog } from '@/entities/session/log-order'
import {
  type SessionStatusMeta,
  type SessionTone,
  sessionStatusMeta,
} from '@/entities/session/status'
import type { Session } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'

// Tone tints the seal's border + number only — never its background. The base
// keeps an OPAQUE fill (grimório bg) so the rail line stays hidden behind every
// node; a translucent tint would let the line show through (live-node bug).
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

export type SessionLogProps = {
  sessions: Session[]
  campaignId: number
  /** Trims to the most relevant few (the Visão geral preview); omit for all. */
  limit?: number
}

/**
 * The chronicle log: sessions as entries down a gilt rail, newest first, the
 * live one highlighted (green seal + pulse + "Entrar").
 */
export function SessionLog(props: SessionLogProps) {
  const shown = () => {
    const ordered = orderSessionsForLog(props.sessions)
    return props.limit ? ordered.slice(0, props.limit) : ordered
  }
  return (
    <ol class="relative space-y-3 py-1">
      <span aria-hidden="true" class="absolute bottom-3 left-4 top-3 w-px bg-grimorio-iron/70" />
      <For each={shown()}>
        {(session) => <SessionLogEntry session={session} campaignId={props.campaignId} />}
      </For>
    </ol>
  )
}

function SessionLogEntry(props: { session: Session; campaignId: number }) {
  const meta = () => sessionStatusMeta(props.session.status)
  const isLive = () => meta().tone === 'live'
  return (
    <li class="relative pl-11">
      <SessionSeal number={props.session.sessionNumber} tone={meta().tone} />
      <Link
        to="/campaigns/$id/sessions/$sid"
        params={{ id: String(props.campaignId), sid: String(props.session.id) }}
        class={cn(
          'flex items-center justify-between gap-3 rounded-none border bg-grimorio-panel p-3 transition-colors',
          isLive()
            ? 'border-[color:var(--hp-full)]/60 bg-[color:var(--hp-full)]/[0.06] hover:border-[color:var(--hp-full)]'
            : 'border-grimorio-iron hover:border-grimorio-gold',
        )}
      >
        <div class="min-w-0">
          <p class="truncate font-medium text-foreground">
            Sessão {props.session.sessionNumber}
            <Show when={props.session.title}>
              {(title) => <span class="text-muted-foreground"> — {title()}</span>}
            </Show>
          </p>
          <p class="text-xs text-muted-foreground">
            {new Date(props.session.createdAt).toLocaleDateString('pt-BR')}
          </p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <StatusPill meta={meta()} />
          <Show when={isLive()}>
            <span class="hidden items-center gap-1 rounded-none bg-[color:var(--hp-full)]/15 px-2 py-1 text-xs font-semibold text-[color:var(--hp-full)] sm:inline-flex">
              Entrar
              <ChevronRight aria-hidden="true" class="size-3.5" />
            </span>
          </Show>
        </div>
      </Link>
    </li>
  )
}

/** The rail node — the session's number as a wax-seal disc, tinted by tone. */
function SessionSeal(props: { number: number; tone: SessionTone }) {
  return (
    <span
      aria-hidden="true"
      class={cn(
        'absolute left-0 top-2.5 flex size-8 items-center justify-center rounded-full border bg-grimorio-bg font-heading text-sm',
        SEAL_TONE[props.tone],
      )}
    >
      {props.number}
    </span>
  )
}

/** Status chip: live pulses green, planned is gilt, ended recedes to iron. */
function StatusPill(props: { meta: SessionStatusMeta }) {
  return (
    <span
      class={cn(
        'inline-flex items-center gap-1.5 rounded-none border px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wider',
        PILL_TONE[props.meta.tone],
      )}
    >
      <Show when={props.meta.tone === 'live'}>
        <span class="size-1.5 animate-pulse rounded-full bg-[color:var(--hp-full)] motion-reduce:animate-none" />
      </Show>
      {props.meta.label}
    </span>
  )
}
