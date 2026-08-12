import { ATTRIBUTE_ABBR, type AttributeKey } from '@tormenta20/t20-data'
import { ChevronRight } from 'lucide-solid'
import { For, type JSX, Show, createSignal } from 'solid-js'
import { AbilityLine } from '@/shared/ui/ability-line'
import { Badge } from '@/shared/ui/badge'
import { cn } from '@/shared/lib/utils'
import type { GrantLine } from './grant-helpers'
import { classGrant, signed } from './grant-helpers'

/**
 * Inline "what this pick grants" box, rendered live under a picker in the
 * Forja so the player sees a race/class/origin's deltas and abilities before
 * committing — no hidden bonuses.
 */
export function GrantBox(props: { title: string; class?: string; children: JSX.Element }) {
  return (
    <div
      class={cn(
        'space-y-2 rounded-md border border-grimorio-iron bg-muted/20 p-3 text-sm',
        props.class,
      )}
    >
      <p class="font-heading text-[11px] uppercase tracking-[0.16em] text-grimorio-gold">
        {props.title}
      </p>
      {props.children}
    </div>
  )
}

export type AbilityDisclosureProps = {
  /** Plural noun, e.g. "habilidades". */
  label: string
  /** Singular for a single line, e.g. "habilidade". Falls back to `label`. */
  singular?: string
  lines: GrantLine[]
  defaultOpen?: boolean
}

/**
 * Collapsed ability list — `▸ N habilidades`, expanding on click. The prose is
 * reference, not a decision input, so it stays folded and the step keeps its
 * height for the choices themselves.
 */
export function AbilityDisclosure(props: AbilityDisclosureProps) {
  const [open, setOpen] = createSignal(props.defaultOpen ?? false)
  const noun = () =>
    props.lines.length === 1 ? (props.singular ?? props.label) : props.label

  return (
    <Show when={props.lines.length > 0}>
      <div class="space-y-1.5">
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          aria-expanded={open()}
          class="flex items-center gap-1 text-[11px] font-semibold uppercase tracking-wide text-muted-foreground hover:text-foreground"
        >
          <ChevronRight
            class={cn('size-3 transition-transform', open() && 'rotate-90')}
            aria-hidden="true"
          />
          {props.lines.length} {noun()}
        </button>
        <Show when={open()}>
          <ul class="space-y-1.5">
            <For each={props.lines}>
              {(line) => <AbilityLine name={line.name} description={line.description} />}
            </For>
          </ul>
        </Show>
      </div>
    </Show>
  )
}

/**
 * Signed attribute-delta chips (`+2 CON`, `−1 DES`). A penalty takes the
 * outline variant so it never reads as a bonus at a glance.
 */
export function DeltaBadges(props: { deltas: Partial<Record<AttributeKey, number>> }) {
  const entries = () =>
    (Object.entries(props.deltas) as [AttributeKey, number][]).filter(([, v]) => v !== 0)

  return (
    <Show when={entries().length > 0}>
      <div class="flex flex-wrap gap-1.5">
        <For each={entries()}>
          {([key, amount]) => (
            <Badge variant={amount < 0 ? 'outline' : 'secondary'} class="font-mono">
              {signed(amount)} {ATTRIBUTE_ABBR[key]}
            </Badge>
          )}
        </For>
      </div>
    </Show>
  )
}

export type ClassGrantLinesProps = { className: string; level: number }

/**
 * What a class is worth through a level: its vitals line and the powers it
 * grants automatically. Rendered inside the chosen-class panel, next to that
 * class's own level control, so raising the level visibly buys something.
 */
export function ClassGrantLines(props: ClassGrantLinesProps) {
  const grant = () => classGrant(props.className, props.level)

  return (
    <>
      <Show when={grant().vitals}>
        {(vitals) => (
          <p class="font-mono text-[11px] text-muted-foreground">
            PV {vitals().pvInicial} inicial (+{vitals().pvPerLevel}/nível) · PM +
            {vitals().mpPerLevel}/nível
          </p>
        )}
      </Show>
      <AbilityDisclosure
        label="habilidades automáticas"
        singular="habilidade automática"
        lines={grant().powers}
      />
    </>
  )
}
