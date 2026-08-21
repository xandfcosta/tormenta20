import { useQueryClient } from '@tanstack/solid-query'
import { FLAG_ACTIVATIONS } from '@/shared/rules/flag-activations'
import { Flame } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { resolveStanceDisplay } from '@/entities/character/conditional-display'
import { type ConditionalEntry, allConditionals } from '@/entities/character/derived'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { useStanceActivations } from '@/shared/stores/stance-activation-context'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import {
  abbreviateConditionalTarget,
  describeConditionalTarget,
} from './conditional-target-label'
import { signed } from './signed'
import { usePowerActions } from './use-power-actions'

export type StanceGroup = { flag: string; entries: ConditionalEntry[] }

/**
 * Power-stance flag groups (FLAG_ACTIVATIONS) that are fully ON. The on-switch
 * lives in the Poderes block; only ACTIVE stances render here.
 */
export function activeStanceGroups(entries: readonly ConditionalEntry[]): StanceGroup[] {
  const byFlag = new Map<string, ConditionalEntry[]>()
  for (const entry of entries) {
    const flag = entry.effect.flag
    if (!flag || !FLAG_ACTIVATIONS[flag]) continue
    byFlag.set(flag, [...(byFlag.get(flag) ?? []), entry])
  }
  return [...byFlag]
    .filter(([, list]) => list.every((entry) => entry.active))
    .map(([flag, list]) => ({ flag, entries: list }))
}

/**
 * "Atq/Dano/Fort/Von +3" — the ENGINE's non-stacking resolution over the flag
 * group, so a Bárbaro 6 reads +3 and not also the superseded +2 tier, then
 * grouped by amount.
 */
export function stanceSummary(group: StanceGroup): string {
  const kept = resolveStanceDisplay(
    group.entries.map((entry) => ({
      target: entry.effect.target,
      bonusType: entry.effect.bonusType,
      amount: entry.effect.amount,
    })),
  )
  const byAmount = new Map<number, string[]>()
  for (const row of kept) {
    byAmount.set(row.amount, [
      ...(byAmount.get(row.amount) ?? []),
      abbreviateConditionalTarget(row.target),
    ])
  }
  return [...byAmount]
    .map(([amount, labels]) => `${labels.join('/')} ${signed(amount)}`)
    .join(' · ')
}

/**
 * "Posturas ativas" — one card per running stance: what it grants, what was
 * paid (including the display-only stepper extra) and an Encerrar exit. Hidden
 * entirely while no stance is up.
 */
export function StancesSection(props: { character: Character }) {
  const conditionals = useConditionals()
  const groups = createMemo(() =>
    activeStanceGroups(
      allConditionals(props.character, conditionals.active(props.character.id)),
    ),
  )

  return (
    <Show when={groups().length > 0}>
      <section class="rounded-none border border-grimorio-iron p-3">
        <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
          Posturas ativas
        </h3>
        <ul class="mt-2 space-y-1">
          <For each={groups()}>
            {(group) => <StanceCard group={group} character={props.character} />}
          </For>
        </ul>
      </section>
    </Show>
  )
}

/**
 * One-line stance summary — the old card listed every tier, including the
 * superseded ones. Tapping the line (not Encerrar) expands the audited
 * per-target breakdown.
 */
function StanceCard(props: { group: StanceGroup; character: Character }) {
  const queryClient = useQueryClient()
  const stanceActivations = useStanceActivations()
  const conditionals = useConditionals()
  const actions = usePowerActions()
  const [expanded, setExpanded] = createSignal(false)

  const activation = () => FLAG_ACTIVATIONS[props.group.flag]
  // Falls back to the base cost for a stance toggled before the payment record
  // existed (legacy toggle, cleared storage).
  const pmPaid = () =>
    stanceActivations.paidFor(props.character.id, props.group.flag)?.pmPaid ??
    activation().pmCost
  const tempHp = createMemo(() =>
    props.group.flag === 'furia'
      ? computedSheetFor(props.character, conditionals.active(props.character.id)).tempHpFuria
      : null,
  )

  return (
    <li class="rounded-none border border-violet-500/25 bg-violet-950/30 px-2 py-1.5">
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setExpanded(!expanded())}
          aria-expanded={expanded()}
          class="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-left"
        >
          <Flame aria-hidden="true" class="size-3.5 shrink-0 text-violet-300" />
          <span class="shrink-0 text-sm font-medium text-foreground">{activation().name}</span>
          <span class="min-w-0 truncate text-[11px] text-muted-foreground">
            · {stanceSummary(props.group)}
            {(tempHp()?.total ?? 0) > 0 ? ` · PV temp +${tempHp()?.total}` : ''}
            {` · ${pmPaid()} PM`}
          </span>
        </button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          class="h-11 shrink-0 px-3 text-xs text-red-400 hover:bg-red-950/40 sm:h-6 sm:px-2 sm:text-[11px]"
          aria-label={`Encerrar ${activation().name}`}
          onClick={() =>
            void actions(queryClient, props.character).deactivateStance(props.group.flag)
          }
        >
          Encerrar
        </Button>
      </div>
      <Show when={expanded()}>
        <ul class="ml-5 mt-1 space-y-0.5 text-[11px]">
          <For each={props.group.entries}>
            {(entry) => (
              <li class="flex items-center justify-between gap-2">
                <span class="truncate text-muted-foreground">
                  {describeConditionalTarget(entry.effect.target)}
                </span>
                <span
                  class={cn(
                    'shrink-0 font-mono font-semibold',
                    entry.effect.amount >= 0 ? 'text-emerald-300' : 'text-red-300',
                  )}
                >
                  {signed(entry.effect.amount)}
                </span>
              </li>
            )}
          </For>
          <Show when={(tempHp()?.total ?? 0) > 0}>
            <li class="flex items-center justify-between gap-2">
              <span class="truncate text-muted-foreground">
                Alma de Bronze — PV temporários (nível + For)
              </span>
              <span class="shrink-0 font-mono font-semibold text-emerald-300">
                +{tempHp()?.total}
              </span>
            </li>
          </Show>
        </ul>
      </Show>
    </li>
  )
}
