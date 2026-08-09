import { Swords } from 'lucide-react'
import { cn } from '@/shared/lib/utils'

/** Whose turn it is, from the player's point of view. */
export type LiveTurnState =
  | { kind: 'mine' }
  | { kind: 'other'; label: string }
  | { kind: 'idle' }

/**
 * "Modo Jogo" banner (ALE-30): a loud, always-visible strip that tells the
 * player they're in a LIVE session — distinct from just editing the sheet. It
 * pulses on "AO VIVO", shows the round, and lights up primary with "SUA VEZ"
 * the moment the player's combatant is active. Meant to sit sticky above the
 * player's sheet inside the match shell.
 */
export function LiveSessionBanner({
  sessionNumber,
  round,
  turn,
}: {
  sessionNumber: number
  round: number
  turn: LiveTurnState
}) {
  const mine = turn.kind === 'mine'
  return (
    <div
      className={cn(
        'sticky top-0 z-10 flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b px-3 py-2 backdrop-blur transition-colors sm:px-4',
        mine
          ? 'border-[color:var(--primary)]/60 bg-[color:var(--primary)]/10'
          : 'border-border/60 bg-card/70',
      )}
    >
      <p className="flex items-center gap-2">
        <LivePip />
        <span className="font-display text-xs font-semibold uppercase tracking-widest text-[color:var(--hp-full)]">
          Ao vivo
        </span>
        <span className="text-sm text-muted-foreground">
          · Sessão {sessionNumber}
        </span>
      </p>
      <div className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground">Rodada {round}</span>
        {mine ? (
          <span className="flex animate-pulse items-center gap-1 rounded-md bg-[color:var(--primary)] px-2 py-0.5 text-xs font-bold uppercase tracking-wide text-[color:var(--primary-foreground)]">
            <Swords className="size-3.5" /> Sua vez
          </span>
        ) : turn.kind === 'other' ? (
          <span className="truncate text-muted-foreground">
            · Vez de {turn.label}
          </span>
        ) : (
          <span className="text-muted-foreground">· Aguardando iniciativa</span>
        )}
      </div>
    </div>
  )
}

/** Pulsing "live" dot (ping halo + solid core), in the full-HP green. */
function LivePip() {
  return (
    <span className="relative flex size-2.5">
      <span className="absolute inline-flex size-full animate-ping rounded-full bg-[color:var(--hp-full)] opacity-75" />
      <span className="relative inline-flex size-2.5 rounded-full bg-[color:var(--hp-full)]" />
    </span>
  )
}
