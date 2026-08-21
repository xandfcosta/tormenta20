import { For, Show, type Component } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/shared/ui/dialog'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'
import type { StatRow } from './stat-rows'

/**
 * A HUD stat tile and the breakdown behind it. The React app had this twice —
 * `CombatBox` and `MagicBox`, ~100 near-identical lines each differing only in
 * palette — so a fix to one drifted from the other. Here the palette is a
 * `tone` and the structure exists once.
 */
export type StatTone = 'combat' | 'magic'

const TONE: Record<StatTone, { trigger: string; label: string; value: string; total: string }> = {
  combat: {
    trigger: 'border-destructive/50 hover:bg-destructive/10',
    label: 'text-destructive/80',
    value: 'text-foreground',
    total: 'border-destructive/40 bg-destructive/10',
  },
  magic: {
    trigger: 'border-arcane/40 hover:bg-arcane/10',
    label: 'text-arcane-ink/80',
    value: 'text-arcane-ink',
    total: 'border-arcane/40 bg-arcane/10',
  },
}

export type StatBoxProps = {
  label: string
  /** Full name for the dialog when the tile label is abbreviated ("Atq CaC"). */
  dialogTitle?: string
  value: number
  rows: StatRow[]
  icon: Component<{ class?: string }>
  tone?: StatTone
  /** Render the value with an explicit sign (+3 / −1). */
  signedValue?: boolean
  /** Small companion line under the value (e.g. "RD 4"). */
  sub?: string
  /** A titled section after the total: values that RELATE to the stat but do
   *  not sum into it (the RD sources under Defesa). */
  extra?: { title: string; rows: StatRow[] }
}

export function StatBox(props: StatBoxProps) {
  const tone = () => TONE[props.tone ?? 'combat']
  const display = () => (props.signedValue ? signed(props.value) : String(props.value))

  return (
    <Dialog>
      <DialogTrigger
        as="button"
        type="button"
        aria-label={`${props.label} ${display()}`}
        class={cn(
          'relative flex cursor-pointer flex-col items-center rounded-none border-2 bg-grimorio-panel p-2 text-center outline-none transition-colors',
          tone().trigger,
        )}
      >
        <span
          class={cn(
            'flex items-center gap-1 text-4xs font-bold uppercase tracking-widest',
            tone().label,
          )}
        >
          <Dynamic component={props.icon} class="size-3.5" />
          {props.label}
        </span>
        <span class={cn('mt-0.5 text-2xl font-bold leading-none', tone().value)}>{display()}</span>
        <Show when={props.sub}>
          {(sub) => (
            <span
              class={cn('text-3xs font-semibold uppercase tracking-widest', tone().label)}
            >
              {sub()}
            </span>
          )}
        </Show>
      </DialogTrigger>

      <DialogContent class="w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2 font-heading uppercase tracking-wide text-grimorio-gold">
            <Dynamic component={props.icon} class="size-3.5" />
            {props.dialogTitle ?? props.label}
          </DialogTitle>
        </DialogHeader>
        <div class="space-y-2 text-sm">
          <StatRowList rows={props.rows} />
          <div
            class={cn(
              'flex items-center justify-between rounded-none border px-3 py-2',
              tone().total,
            )}
          >
            <span class={cn('text-xs uppercase tracking-widest', tone().label)}>Total</span>
            <span class={cn('font-mono text-2xl font-bold', tone().value)}>{display()}</span>
          </div>
          <Show when={props.extra}>
            {(extra) => (
              <div class="space-y-1">
                <p class={cn('text-xs font-bold uppercase tracking-widest', tone().label)}>
                  {extra().title}
                </p>
                <StatRowList rows={extra().rows} />
              </div>
            )}
          </Show>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function StatRowList(props: { rows: StatRow[] }) {
  return (
    <ul class="space-y-1">
      <For each={props.rows}>{(row) => <StatRowLine row={row} />}</For>
    </ul>
  )
}

/**
 * One breakdown line: source + amount, with the modifier's note (the WHY —
 * "desbalanceada: -2 em ataque") as a dim sub-line, so a row explains itself
 * instead of showing a bare item name.
 */
export function StatRowLine(props: { row: StatRow }) {
  return (
    <li class={cn('border-b border-border pb-1', props.row.muted && 'text-muted-foreground')}>
      <div class="flex items-center justify-between gap-2">
        <span class="truncate">{props.row.label}</span>
        <span class="shrink-0 font-mono">{signed(props.row.amount)}</span>
      </div>
      {/* Wrap, never truncate: a nowrap note becomes the grid's min-content and
          inflates the whole dialog past its max-width. */}
      <Show when={props.row.note}>
        {(note) => <p class="text-2xs leading-snug text-muted-foreground">{note()}</p>}
      </Show>
    </li>
  )
}
