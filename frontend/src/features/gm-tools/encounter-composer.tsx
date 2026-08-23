import { Trash2 } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import { cn } from '@/shared/lib/utils'
import { encounterXp } from '@/shared/rules/xp'
import { NumberInput } from '@/shared/ui/number-input'
import { SectionLabel } from '@/shared/ui/section-label'
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
        <SectionLabel>
          O grupo
        </SectionLabel>
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

      {/* O VEREDITO é o bloco principal, e isso é a issue (ALE-170): esta
          ferramenta existe para responder "esse encontro é duro demais?", e a
          resposta era uma faixa de 14px — mais discreta que o botão de
          adicionar criatura logo abaixo. Agora o ND é o número grande e a
          dificuldade vem ao lado com o peso que ela merece; o XP fica
          secundário porque ele é consequência, não decisão. */}
      <div class="flex flex-wrap items-center gap-x-8 gap-y-3 rounded-sm border border-grimorio-iron bg-muted/20 px-4 py-3">
        <div class="flex items-baseline gap-3">
          <SectionLabel as="span" class="text-3xs">
            ND do encontro
          </SectionLabel>
          <span class="font-mono text-4xl leading-none text-grimorio-gold tabular-nums">
            {formatNd(round2(nd()))}
          </span>
        </div>
        <div class="flex items-baseline gap-2">
          <SectionLabel as="span" class="text-3xs">
            Dificuldade
          </SectionLabel>
          <span
            class="font-mono text-xl leading-none"
            style={{ color: TONE_COLOR[difficulty().tone] }}
          >
            {difficulty().label}
          </span>
        </div>
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
      <SectionLabel
        for={props.id}
       
       as="label" class="text-3xs block">
        {props.label}
      </SectionLabel>
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
      <SectionLabel as="span" class="text-3xs">
        {props.label}
      </SectionLabel>
      <span
        class={cn('font-mono text-sm', !props.color && 'text-grimorio-gold')}
        style={props.color ? { color: props.color } : undefined}
      >
        {props.value}
      </span>
    </p>
  )
}
