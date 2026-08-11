import { useQueryClient } from '@tanstack/solid-query'
import { Minus, Plus } from 'lucide-solid'
import { Show, createMemo, createSignal, onCleanup } from 'solid-js'
import { tempHpPool } from '@/entities/character/temp-hp-pool'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { cn } from '@/shared/lib/utils'
import { AttributesGrid } from './attributes-grid'
import { ConditionPips } from './condition-pips'
import { MobileDefChip } from './mobile-def-chip'
import { type TempPoolControl, ResourceAdjustDialog } from './resource-adjust-dialog'
import { ClassBadges, SheetIdentityText } from './sheet-identity'
import { createVitalActions } from './vital-mutations'

/**
 * The HUD pinned to the bottom of the sheet on both layouts: a player
 * "nameplate" — square portrait beside the identity stacked over the PV/PM bars
 * — plus the attribute boxes from tablet up.
 *
 * The React version wrapped this in `memo()`; its comment explains that doing so
 * is what collapsed the ~160ms-per-block-switch floor that survived memoizing
 * the panels. There is no wrapper here: a switch touches only what reads the
 * signal that changed, and `e2e/bench-tabs.mjs` is what settles whether the
 * floor came back (ALE-90).
 */
export function CharacterHud(props: { character: Character; class?: string }) {
  const queryClient = useQueryClient()
  const conditionals = useConditionals()
  const active = createMemo(() => conditionals.active(props.character.id))

  // Created ONCE (Solid runs this body a single time): the actions own the
  // debounce timer and the rollback snapshot of the whole click burst.
  const vitals = createVitalActions(queryClient, () => props.character)

  const pool = createMemo(() => tempHpPool(props.character))

  /**
   * Lowering PV is DAMAGE, not a vitals write: the server routes it through the
   * temp-PV pool first (one atomic request). Raising it is plain healing.
   */
  const applyHp = (next: number) => {
    const damage = props.character.hpCurrent - next
    if (damage <= 0) {
      vitals.setHp(next)
      return
    }
    void vitals.applyDamage(damage)
  }

  return (
    <div class={cn('border-t border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2 sm:px-4', props.class)}>
      {/* Below lg the attribute cluster stacks BELOW the nameplate: side by
          side it stretched the square portrait, which squeezed the PV/PM row
          until its −/+ buttons slid under the cluster and stopped being
          tappable. */}
      <div class="flex flex-col gap-3 lg:flex-row lg:items-stretch lg:gap-4">
        <div class="flex min-w-0 items-stretch gap-3 lg:w-[34rem] lg:flex-none">
          {/* `h-auto w-24 self-stretch`, not `aspect-square`: aspect-ratio
              cannot derive a width from a flex-stretched height, so the
              portrait collapsed to the width of its initials. */}
          <CharacterPortrait
            name={props.character.name}
            size="sm"
            class="h-auto w-24 self-stretch rounded-sm border-2 border-grimorio-iron text-4xl"
          />
          <div class="flex min-w-0 flex-1 flex-col gap-1.5">
            <div class="flex items-start justify-between gap-2">
              <SheetIdentityText character={props.character} activeConditionals={active()} />
              <div class="flex shrink-0 items-center gap-1.5">
                <MobileDefChip
                  character={props.character}
                  activeConditionals={active()}
                  class="md:hidden"
                />
                <LevelBadge character={props.character} />
              </div>
            </div>
            {/* Class badges and condition pips share ONE row — a dedicated
                conditions row doubled the nameplate height. */}
            <div class="flex flex-wrap items-center gap-1">
              <ClassBadges character={props.character} />
              <ConditionPips character={props.character} mini />
            </div>
            <div class="mt-auto flex flex-col gap-1">
              <HudVital
                label="Vida"
                current={props.character.hpCurrent}
                max={props.character.hpMax}
                kind="hp"
                onSet={applyHp}
                onDamage={(amount) => void vitals.applyDamage(amount)}
                temp={pool().total}
                tempTitle={pool()
                  .slices.map((slice) => slice.label)
                  .join(', ')}
                tempPool={{
                  total: pool().total,
                  onSetManual: (value) => void vitals.setManualTempHp(value),
                }}
              />
              <HudVital
                label="Mana"
                current={props.character.mpCurrent}
                max={props.character.mpMax}
                kind="mp"
                onSet={vitals.setMp}
              />
            </div>
          </div>
        </div>

        {/* From md the attribute boxes ride along — this IS the Vitais content,
            which is why the phone layout keeps its own Vitais section and these
            widths do not. */}
        <div class="hidden min-w-0 flex-1 flex-col justify-center gap-1.5 md:flex">
          <AttributesGrid
            character={props.character}
            activeConditionals={active()}
            class="grid-cols-6"
          />
        </div>
      </div>
    </div>
  )
}

/** Total level, read-only. The multiclass stepper lands with the rest of the
 *  level mutation (ALE-90, fatia seguinte). */
function LevelBadge(props: { character: Character }) {
  return (
    <span class="shrink-0 rounded-sm border border-grimorio-gold/50 px-2 py-0.5 font-mono text-xs font-bold text-grimorio-gold">
      Nv {props.character.level}
    </span>
  )
}

/** HP fill token by ratio — the COLOR, not just the width, says "how bad". */
export function hpFillVar(percent: number): string {
  if (percent <= 25) return '--hp-critical'
  if (percent <= 50) return '--hp-hurt'
  return '--hp-full'
}

type HudVitalProps = {
  label: string
  current: number
  max: number
  kind: 'hp' | 'mp'
  onSet: (next: number) => void
  /** Routes "−" through the atomic damage endpoint (temp-first). */
  onDamage?: (amount: number) => void
  /** Debitable temp-PV pool shown as "+N". */
  temp?: number
  tempTitle?: string
  tempPool?: TempPoolControl
}

/**
 * One PV/PM row: label · decay-colored bar · current/max · −/+ · bulk edit.
 * A real `progressbar`, so the number is readable by assistive tech and by the
 * E2E suite.
 */
function HudVital(props: HudVitalProps) {
  const percent = () =>
    props.max > 0 ? Math.max(0, Math.min(100, (props.current / props.max) * 100)) : 0
  const fillVar = () => (props.kind === 'hp' ? hpFillVar(percent()) : '--mp-arcane')
  const temp = () => props.temp ?? 0
  const delta = createVitalDelta(() => props.current)
  // Shift-click steps ±5 — combat deltas are rarely 1.
  const stepOf = (event: MouseEvent) => (event.shiftKey ? 5 : 1)

  return (
    <div class="flex items-center gap-1.5 sm:gap-2">
      <span
        class="w-9 shrink-0 text-[10px] font-bold uppercase tracking-wider"
        style={{ color: `var(${fillVar()})` }}
      >
        {props.label}
      </span>
      {/* − sits on the far side of + so a greasy thumb never heals when it
          meant to hurt. Damage goes out UNCLAMPED: the server routes it
          temp-first, so a shift−5 at 3 PV still drains 5 from the pool. */}
      <Button
        type="button"
        variant="outline"
        size="icon"
        class="size-9 shrink-0 lg:size-6"
        disabled={props.current <= 0 && temp() <= 0}
        onClick={(event) =>
          props.onDamage
            ? props.onDamage(stepOf(event))
            : props.onSet(Math.max(0, props.current - stepOf(event)))
        }
        aria-label={`Reduzir ${props.label} (shift: 5)`}
      >
        <Minus aria-hidden="true" class="size-4 lg:size-3" />
      </Button>

      <div
        role="progressbar"
        aria-label={props.label}
        aria-valuenow={props.current}
        aria-valuemin={0}
        aria-valuemax={props.max}
        class="relative h-3.5 min-w-8 flex-1 overflow-hidden rounded-full border border-border bg-muted lg:h-2.5"
      >
        <div
          class="h-full transition-[width,background-color] duration-500 ease-out"
          style={{ width: `${percent()}%`, 'background-color': `var(${fillVar()})` }}
        />
      </div>

      <span class="relative shrink-0 font-mono text-base tabular-nums lg:text-xs">
        <span class="font-bold">{props.current}</span>
        <span class="text-muted-foreground">/{props.max}</span>
        <Show when={temp() > 0}>
          <span class="ml-1 font-bold text-emerald-400" title={props.tempTitle ?? 'PV temporários'}>
            +{temp()}
          </span>
        </Show>
        <Show when={delta() !== null}>
          <span
            class={cn(
              'absolute -top-4 right-0 text-[10px] font-bold',
              (delta() ?? 0) < 0
                ? 'text-[color:var(--hp-critical)]'
                : 'text-[color:var(--hp-full)]',
            )}
          >
            {(delta() ?? 0) > 0 ? `+${delta()}` : delta()}
          </span>
        </Show>
      </span>

      <div class="flex shrink-0 items-center gap-1 lg:gap-0.5">
        <Button
          type="button"
          variant="outline"
          size="icon"
          class="size-9 lg:size-6"
          disabled={props.current >= props.max}
          onClick={(event) => props.onSet(Math.min(props.max, props.current + stepOf(event)))}
          aria-label={`Aumentar ${props.label} (shift: 5)`}
        >
          <Plus aria-hidden="true" class="size-4 lg:size-3" />
        </Button>
        <ResourceAdjustDialog
          label={props.label}
          current={props.current}
          max={props.max}
          onSetCurrent={props.onSet}
          onDamage={props.onDamage}
          tempPool={props.tempPool}
          triggerClass="size-9 lg:size-6"
        />
      </div>
    </div>
  )
}

/**
 * Lingering delta chip: ANY change to the value — the player's own click or an
 * external one (the GM's socket damage, an expired effect) — shows "+X/−X" for
 * a few seconds, so someone who looked away still learns WHAT changed instead
 * of only seeing a different number.
 */
export function createVitalDelta(current: () => number) {
  const [delta, setDelta] = createSignal<number | null>(null)
  let previous = current()
  let timer: ReturnType<typeof setTimeout> | undefined

  createMemo(() => {
    const value = current()
    const diff = value - previous
    previous = value
    if (diff === 0) return
    setDelta(diff)
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => setDelta(null), 3000)
  })

  onCleanup(() => timer && clearTimeout(timer))
  return delta
}
