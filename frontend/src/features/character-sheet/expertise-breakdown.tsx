import { Lock } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import type { BreakdownContribution } from '@/shared/lib/computed-sheet-v2'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { signed } from './signed'

export type ExpertiseBreakdownProps = {
  name: string
  total: number
  /** Trained-only perícia the character has not trained — unusable. */
  locked: boolean
  halfLevel: number
  attrAbbr: string
  attrMod: number
  trainBonus: number
  itemBonus: number
  contributions: BreakdownContribution[]
  /** The row itself; it carries the `DialogTrigger`s that open this. */
  children: JSX.Element
}

/**
 * Where a perícia's number comes from: ½ level + attribute + training + each
 * item/effect that touched it, then the total. The whole point of the sheet is
 * that a player can audit a roll, so every contribution names its source.
 */
export function ExpertiseBreakdown(props: ExpertiseBreakdownProps) {
  return (
    <Dialog>
      {props.children}
      <DialogContent class="w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6">
        <DialogHeader>
          <DialogTitle class="font-heading uppercase tracking-wide text-grimorio-gold">
            {props.name}
          </DialogTitle>
        </DialogHeader>
        <div class="space-y-1 text-sm">
          <BreakdownRow label="½ nível" value={props.halfLevel} />
          <BreakdownRow label={`Atributo (${props.attrAbbr})`} value={props.attrMod} />
          <BreakdownRow label="Treino" value={props.trainBonus} />
          <BreakdownRow label="Outros" value={props.itemBonus} />
          <For each={props.contributions}>
            {(contribution) => (
              <BreakdownRow
                label={contribution.source}
                value={contribution.amount}
                note={contribution.note}
                indented
              />
            )}
          </For>
          <div class="mt-2 flex items-center justify-between rounded-none border border-grimorio-iron bg-grimorio-panel-raised px-3 py-2">
            <span class="text-xs uppercase tracking-widest text-muted-foreground">Total</span>
            <span
              class="flex items-center gap-1.5 font-mono text-2xl font-bold"
              classList={{
                'text-muted-foreground/50': props.locked,
                'text-grimorio-gold': !props.locked,
              }}
            >
              <Show when={props.locked}>
                <Lock aria-hidden="true" class="size-4 text-warning" />
              </Show>
              {signed(props.total)}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function BreakdownRow(props: {
  label: string
  value: number
  /** Modifier note — the WHY ("desbalanceada: -2 em ataque"). */
  note?: string
  indented?: boolean
}) {
  return (
    <div
      class="border-b border-grimorio-iron/60 py-1"
      classList={{ 'pl-4 text-xs opacity-80': props.indented }}
    >
      <div class="flex items-center justify-between gap-2">
        <span class="truncate">{props.label}</span>
        <span class="shrink-0 font-mono">{signed(props.value)}</span>
      </div>
      {/* Wraps, never truncates: a nowrap note becomes min-content and can
          inflate the dialog past its max-width. */}
      <Show when={props.note}>
        {(note) => <p class="text-3xs leading-snug text-muted-foreground">{note()}</p>}
      </Show>
    </div>
  )
}
