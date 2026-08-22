import { ChevronUp, Swords } from 'lucide-solid'
import { type JSX, Show, createSignal } from 'solid-js'
import { createMediaQuery } from '@/shared/lib/media-query'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'

/**
 * The session's side rail, which changes shape with the viewport. On desktop
 * it is an aside beside the main surface; on a phone the main surface (tracker
 * or sheet) owns the screen and the rail collapses into a fixed bottom bar
 * showing a live peek, with the full controls one tap away.
 *
 * Rendered ONCE either way — a media query, not duplicated DOM — so the
 * interactive cards inside keep a single mount.
 */
export function MatchRail(props: {
  title: string
  /** Compact live status for the collapsed phone bar. */
  peek?: JSX.Element
  children: JSX.Element
}) {
  const isDesktop = createMediaQuery('(min-width: 1024px)')

  return (
    <Show
      when={isDesktop()}
      fallback={
        <div
          class="fixed inset-x-0 bottom-0 z-40 flex items-center gap-3 border-t border-grimorio-iron bg-grimorio-panel px-3 py-2"
          // Keeps the `py-2` floor and grows to clear the home indicator when
          // the app runs edge-to-edge (`viewport-fit=cover`).
          style={{ 'padding-bottom': 'max(0.5rem, env(safe-area-inset-bottom))' }}
        >
          <div class="min-w-0 flex-1 truncate text-sm text-muted-foreground">{props.peek}</div>
          <MatchControls title={props.title}>{props.children}</MatchControls>
        </div>
      }
    >
      <aside class="max-h-full space-y-4 overflow-y-auto">{props.children}</aside>
    </Show>
  )
}

/**
 * The rail's controls behind a trigger, separate from the fixed bar: the
 * player view folds this into the sheet's own bottom bar so a phone shows one
 * bar, not two stacked.
 */
export function MatchControls(props: {
  title: string
  /** Custom trigger; defaults to a chevron + title button. */
  trigger?: (open: () => void) => JSX.Element
  children: JSX.Element
}) {
  const [open, setOpen] = createSignal(false)

  return (
    <>
      {props.trigger?.(() => setOpen(true)) ?? (
        <Button size="sm" variant="outline" class="gap-1.5" onClick={() => setOpen(true)}>
          <ChevronUp aria-hidden="true" class="size-4" />
          {props.title}
        </Button>
      )}
      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="flex max-h-[85vh] w-[calc(100vw-1.5rem)] max-w-lg flex-col gap-0 sm:w-full">
          <DialogHeader>
            <DialogTitle class="font-heading tracking-wide">{props.title}</DialogTitle>
          </DialogHeader>
          <div class="min-h-0 flex-1 space-y-4 overflow-y-auto p-1">{props.children}</div>
        </DialogContent>
      </Dialog>
    </>
  )
}

/** One live line for the collapsed phone bar: round + who is up. */
export function MatchPeek(props: { rt: SessionRealtime }) {
  const active = () => {
    const state = props.rt.state()
    return state.turnIndex >= 0 ? state.initiative[state.turnIndex] : undefined
  }

  return (
    <span class="flex items-center gap-1.5">
      <span class="font-mono tabular-nums">Rodada {props.rt.state().round}</span>
      <Show when={active()}>
        {(entry) => (
          <>
            <Swords aria-hidden="true" class="size-3.5 text-grimorio-crimson-bright" />
            <span class="truncate">{entry().label}</span>
          </>
        )}
      </Show>
    </span>
  )
}
