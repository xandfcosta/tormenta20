import { encounterXp } from '@/shared/rules/xp'
import { Trash2 } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { NumberInput } from '@/shared/ui/number-input'
import {
  type EnrichedGroup,
  encounterDifficulty,
  encounterNd,
} from './encounter'
import { formatNd } from './monster-format'

const TONE_COLOR = {
  calm: 'var(--hp-full)',
  even: 'var(--grimorio-gold)',
  hard: 'var(--hp-hurt)',
  deadly: 'var(--hp-critical)',
} as const

export type EncounterComposerProps = {
  groups: EnrichedGroup[]
  partyLevel: number
  partySize: number
  onPartyLevel: (level: number) => void
  onPartySize: (size: number) => void
  onQuantity: (monsterId: string, quantity: number) => void
  onRemove: (monsterId: string) => void
  /** The "add creature" control — the Mesa opens a panel, the session drawer
   *  reuses its own list, so the composer does not own that decision. */
  addControl: JSX.Element
  /** Extra action under the ledger (e.g. "send to the tracker"). */
  footer?: JSX.Element
}

/**
 * The encounter as it is being built: the party it is aimed at, the creatures
 * in it, and what that adds up to. Shared by the Mesa's Encontros tool and the
 * in-session builder so a GM composes the same way in both.
 */
export function EncounterComposer(props: EncounterComposerProps) {
  const nd = () => encounterNd(props.groups)
  const difficulty = () => encounterDifficulty(nd() - props.partyLevel)
  const xpEach = () =>
    encounterXp({
      nd: nd(),
      partyLevel: props.partyLevel,
      partySize: props.partySize,
      outcome: 'win',
    })

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-3">
      <div class="flex flex-wrap items-end gap-3 rounded-sm border border-grimorio-iron p-3">
        <p class="font-heading text-2xs uppercase tracking-[0.16em] text-muted-foreground">
          O grupo
        </p>
        <PartyField
          id="encounter-party-level"
          label="Nível"
          min={1}
          max={20}
          value={props.partyLevel}
          onChange={props.onPartyLevel}
        />
        <PartyField
          id="encounter-party-size"
          label="Personagens"
          min={1}
          max={8}
          value={props.partySize}
          onChange={props.onPartySize}
        />
      </div>

      <div class="flex flex-wrap items-center gap-x-5 gap-y-2 rounded-sm border border-grimorio-iron bg-muted/20 px-3 py-2">
        <Ledger label="ND do encontro" value={formatNd(round2(nd()))} />
        <Ledger
          label="Dificuldade"
          value={difficulty().label}
          color={TONE_COLOR[difficulty().tone]}
        />
        <Ledger label="XP por personagem" value={xpEach().toLocaleString('pt-BR')} />
      </div>

      {props.addControl}

      <Show
        when={props.groups.length > 0}
        fallback={
          <p class="rounded-sm border border-dashed border-grimorio-iron p-4 text-center text-xs text-muted-foreground">
            Nenhuma criatura no encontro ainda.
          </p>
        }
      >
        <ul class="space-y-1.5">
          <For each={props.groups}>
            {(group) => (
              <li class="flex flex-wrap items-center gap-2 rounded-sm border border-grimorio-iron p-2">
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-xs font-semibold">{group.monster.name}</span>
                  <span class="font-mono text-3xs text-muted-foreground">
                    ND {formatNd(group.monster.nd)} · grupo ND {formatNd(round2(group.groupNd))}
                  </span>
                </span>
                <NumberInput
                  min={1}
                  max={99}
                  value={group.quantity}
                  onChange={(quantity) => props.onQuantity(group.monster.id, quantity)}
                  class="w-20"
                  aria-label={`Quantidade de ${group.monster.name}`}
                  spinnerLabel={group.monster.name}
                />
                <button
                  type="button"
                  aria-label={`Remover ${group.monster.name}`}
                  onClick={() => props.onRemove(group.monster.id)}
                  class="shrink-0 rounded-none p-1 text-muted-foreground transition-colors hover:text-penalty-ink"
                >
                  <Trash2 aria-hidden="true" class="size-4" />
                </button>
              </li>
            )}
          </For>
        </ul>
      </Show>

      <Show when={props.footer}>{(footer) => <div class="pt-1">{footer()}</div>}</Show>
    </div>
  )
}

/** Group ND is fractional by construction (log2) — two decimals is as much
 *  precision as a GM can act on. */
function round2(value: number): number {
  return Math.round(value * 100) / 100
}

function PartyField(props: {
  id: string
  label: string
  min: number
  max: number
  value: number
  onChange: (value: number) => void
}) {
  return (
    <div class="space-y-1">
      <label
        for={props.id}
        class="block font-heading text-3xs uppercase tracking-[0.14em] text-muted-foreground"
      >
        {props.label}
      </label>
      <NumberInput
        id={props.id}
        min={props.min}
        max={props.max}
        value={props.value}
        onChange={props.onChange}
        class="w-24"
        aria-label={`${props.label} do grupo`}
        spinnerLabel={props.label.toLowerCase()}
      />
    </div>
  )
}

function Ledger(props: { label: string; value: string; color?: string }) {
  return (
    <p class="flex items-baseline gap-1.5">
      <span class="font-heading text-3xs uppercase tracking-[0.14em] text-muted-foreground">
        {props.label}
      </span>
      <span
        class={cn('font-mono text-sm', !props.color && 'text-grimorio-gold')}
        style={props.color ? { color: props.color } : undefined}
      >
        {props.value}
      </span>
    </p>
  )
}
