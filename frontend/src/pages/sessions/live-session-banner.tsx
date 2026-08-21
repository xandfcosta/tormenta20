import { Swords } from 'lucide-solid'
import { Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { FieldLabel, SectionLabel } from '@/shared/ui/section-label'

/** Whose turn it is, from the player's point of view. */
export type LiveTurnState =
  | { kind: 'mine' }
  | { kind: 'other'; label: string }
  | { kind: 'idle' }

/**
 * "Modo Jogo" strip (ALE-30): a loud, always-visible band telling the player
 * this is a LIVE session and not plain sheet editing. Sticky above the sheet
 * inside the match shell, and it brightens on the player's own turn.
 */
export function LiveSessionBanner(props: {
  sessionNumber: number
  round: number
  turn: LiveTurnState
}) {
  const mine = () => props.turn.kind === 'mine'

  return (
    <div
      class={cn(
        'sticky top-0 z-10 flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b px-3 py-2 transition-colors sm:px-4',
        mine()
          ? 'border-[color:var(--primary)]/60 bg-[color:var(--primary)]/10'
          : 'border-grimorio-iron bg-grimorio-panel',
      )}
    >
      <p class="flex items-center gap-2">
        <LivePip />
        <SectionLabel as="span" tom="inherit" class="text-xs font-semibold text-[color:var(--hp-full)]">
          Ao vivo
        </SectionLabel>
        <span class="text-sm text-muted-foreground">· Sessão {props.sessionNumber}</span>
      </p>
      <div class="flex items-center gap-2 text-sm">
        <span class="text-muted-foreground">Rodada {props.round}</span>
        <Show
          when={mine()}
          fallback={
            <span class="truncate text-muted-foreground">
              {props.turn.kind === 'other' ? `· Vez de ${props.turn.label}` : '· Aguardando iniciativa'}
            </span>
          }
        >
          <FieldLabel tom="inherit" class="text-xs flex animate-pulse items-center gap-1 rounded-none bg-[color:var(--primary)] px-2 py-0.5 font-bold text-[color:var(--primary-foreground)]">
            <Swords aria-hidden="true" class="size-3.5" /> Sua vez
          </FieldLabel>
        </Show>
      </div>
    </div>
  )
}

/** Pulsing "live" dot (ping halo + solid core), in the full-HP green. */
function LivePip() {
  return (
    <span class="relative flex size-2.5">
      <span class="absolute inline-flex size-full animate-ping rounded-full bg-[color:var(--hp-full)] opacity-75" />
      <span class="relative inline-flex size-2.5 rounded-full bg-[color:var(--hp-full)]" />
    </span>
  )
}
