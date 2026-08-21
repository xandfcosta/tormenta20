import type { ClassPower, GeneralPower, PowerKind } from '@/shared/api/catalog-types'
import { Check, Info } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import type { PrerequisiteCheck } from '@/entities/character/derived'
import { cn } from '@/shared/lib/utils'
import { Popover, PopoverContent, PopoverTrigger } from '@/shared/ui/popover'

/** Kind pill ("combate", "destino"…) shown next to a power's name. */
export function PowerKindBadge(props: { kind: PowerKind }) {
  return (
    <span class="rounded-none bg-muted px-1 py-px text-[9px] uppercase tracking-wide text-muted-foreground">
      {props.kind}
    </span>
  )
}

function LevelBadge(props: { label: string; tone?: 'auto' | 'gate' | 'locked' }) {
  const tone = () => props.tone ?? 'gate'
  return (
    <span
      class={cn(
        'rounded-none px-1 py-px text-[9px] uppercase tracking-wide',
        tone() === 'auto' && 'bg-muted text-muted-foreground',
        tone() === 'gate' && 'border border-border text-muted-foreground',
        tone() === 'locked' && 'bg-destructive text-white',
      )}
    >
      {props.label}
    </span>
  )
}

/** Owned/available toggle; disabled + dimmed when locked and not owned. */
function PowerCheckbox(props: { owned: boolean; disabled?: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={() => props.onToggle()}
      disabled={props.disabled}
      aria-pressed={props.owned}
      aria-label={props.owned ? 'Remover poder' : 'Selecionar poder'}
      class={cn(
        'mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-none border',
        props.owned
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-border hover:bg-muted',
        props.disabled && 'cursor-not-allowed opacity-40',
      )}
    >
      <Show when={props.owned}>
        <Check aria-hidden="true" class="size-3" />
      </Show>
    </button>
  )
}

/** Carries a description off the collapsed row — touch-safe, unlike a title. */
function PowerInfo(props: { text: string }) {
  return (
    <Popover>
      <PopoverTrigger
        as="button"
        type="button"
        aria-label="Ver descrição"
        class="flex size-6 shrink-0 items-center justify-center rounded-none text-muted-foreground hover:bg-muted hover:text-foreground"
      >
        <Info aria-hidden="true" class="size-3.5" />
      </PopoverTrigger>
      {/* No forced side: the popover collision-flips, so it never clips off a
          full-width phone panel. */}
      <PopoverContent class="w-[min(18rem,calc(100vw-2rem))] text-xs leading-snug">
        {props.text}
      </PopoverContent>
    </Popover>
  )
}

/**
 * A general-power row (browse pool). The description stays inline when owned,
 * otherwise it moves behind the info popover to keep the pool scannable.
 * Rendered inside a virtualized list, so it's a `div`, not an `li`.
 */
export function GeneralPowerRow(props: {
  power: GeneralPower
  owned: boolean
  locked?: boolean
  disabled?: boolean
  onToggle: () => void
}) {
  return (
    <div
      class={cn(
        'flex items-start gap-2 rounded-none border border-border p-2',
        props.owned && 'bg-muted',
      )}
    >
      <PowerCheckbox
        owned={props.owned}
        disabled={props.disabled || (props.locked && !props.owned)}
        onToggle={() => props.onToggle()}
      />
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1">
          <p class="text-xs font-semibold text-grimorio-gold">{props.power.name}</p>
          <PowerKindBadge kind={props.power.kind} />
          <Show when={(props.power.minLevel ?? 1) > 1}>
            <LevelBadge label={`≥L${props.power.minLevel}`} />
          </Show>
          <Show when={props.locked && !props.owned}>
            <LevelBadge label="Bloqueado" tone="locked" />
          </Show>
          <Show when={!props.owned}>
            <PowerInfo text={props.power.description} />
          </Show>
        </div>
        <Show when={props.owned}>
          <p class="mt-0.5 text-[11px] leading-snug text-muted-foreground">
            {props.power.description}
          </p>
        </Show>
      </div>
    </div>
  )
}

/**
 * A class-power row: auto-granted powers show their description inline; an
 * unowned elective keeps the prereq line inline (it's the actionable "why
 * locked") but pushes the long description into the info popover.
 */
export function ClassPowerRow(props: {
  power: ClassPower
  owned: boolean
  locked?: boolean
  prereqChecks?: PrerequisiteCheck[]
  disabled?: boolean
  onToggle?: () => void
  /** Use affordance (PowerActionSlot) — only passed for OWNED powers. */
  actionSlot?: JSX.Element
}) {
  const isAuto = () => props.power.grantedAtLevel !== undefined
  const inlineDescription = () => props.owned || isAuto()

  return (
    <li
      class={cn(
        'flex items-start gap-2 rounded-none border border-border p-2',
        props.owned && 'bg-muted',
      )}
    >
      <Show when={!isAuto() && props.onToggle}>
        {(toggle) => (
          <PowerCheckbox
            owned={props.owned}
            disabled={props.disabled || (props.locked && !props.owned)}
            onToggle={toggle()}
          />
        )}
      </Show>
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-1">
          <p class="text-xs font-semibold text-grimorio-gold">{props.power.name}</p>
          <Show when={isAuto()}>
            <LevelBadge label={`L${props.power.grantedAtLevel}`} tone="auto" />
          </Show>
          <Show when={!isAuto() && (props.power.minLevel ?? 1) > 1}>
            <LevelBadge label={`≥L${props.power.minLevel}`} />
          </Show>
          <Show when={props.locked && !props.owned}>
            <LevelBadge label="Bloqueado" tone="locked" />
          </Show>
          <Show when={!inlineDescription()}>
            <PowerInfo text={props.power.description} />
          </Show>
          {/* ml-auto right-aligns on wide rows; flex-wrap drops it to its own
              full line at phone width. */}
          <Show when={props.actionSlot}>
            <span class="ml-auto">{props.actionSlot}</span>
          </Show>
        </div>
        <Show when={!props.owned && (props.prereqChecks?.length ?? 0) > 0}>
          <p class="mt-0.5 text-[10px] leading-snug">
            <span class="font-semibold text-muted-foreground">Requer: </span>
            <For each={props.prereqChecks}>
              {(check, index) => (
                <>
                  <Show when={index() > 0}>
                    <span class="text-muted-foreground">, </span>
                  </Show>
                  <span
                    class={cn(
                      check.prereq.kind === 'note'
                        ? 'text-foreground'
                        : check.met
                          ? 'text-emerald-300'
                          : 'text-red-300',
                    )}
                  >
                    {check.reason}
                  </span>
                </>
              )}
            </For>
          </p>
        </Show>
        <Show when={inlineDescription()}>
          <p class="mt-0.5 text-[11px] leading-snug text-muted-foreground">
            {props.power.description}
          </p>
        </Show>
      </div>
    </li>
  )
}
