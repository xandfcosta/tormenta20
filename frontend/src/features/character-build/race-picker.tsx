import { Search } from 'lucide-solid'
import { For, Show, createSignal } from 'solid-js'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { matchesQuery } from '@/shared/lib/fuzzy-filter'
import { cn } from '@/shared/lib/utils'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { Input } from '@/shared/ui/input'
import { DeformidadeControls } from './deformidade-controls'
import {
  type RaceChoice,
  type RaceChoiceState,
  raceGrant,
  racePending,
  raceSignature,
  racesByTier,
  resolveRaceDeltas,
} from './grant-helpers'
import { AbilityDisclosure, DeltaBadges, GrantBox } from './grant-panels'
import { RaceChoiceControls } from './race-choice-controls'
import { SectionLabel } from '@/shared/ui/section-label'

export type RacePickerProps = {
  /** Race names the backend offers. */
  options: string[]
  value: string[]
  choices: RaceChoiceState
  onChange: (next: string[]) => void
  onChoice: (raceName: string, choice: RaceChoice) => void
}

/**
 * Lineage picker: a searchable, tier-grouped grid of hue-tinted race tiles on
 * the left and the chosen race's detail pinned on the right, so comparing two
 * lineages never costs the catalogue from view.
 *
 * Multi-select is homebrew: the FIRST pick is the mechanical **primary** (its
 * mods always apply); extra races are flavor whose properties only apply when
 * the player opts in — a GM negotiation, never automatic.
 */
export function RacePicker(props: RacePickerProps) {
  const [query, setQuery] = createSignal('')
  const filtered = () => props.options.filter((name) => matchesQuery([name], query()))
  const tiers = () => racesByTier(filtered())

  const toggle = (name: string) =>
    props.onChange(
      props.value.includes(name)
        ? props.value.filter((n) => n !== name)
        : [...props.value, name],
    )

  return (
    // Enquanto NADA foi escolhido, o palco é do catálogo (ALE-171): o
    // painel de detalhe reservava 871px de 1920 — 46% da tela — para dizer
    // "escolha um para ver o que ele concede", enquanto os ladrilhos se
    // espremiam ao lado. É a mesma regra que a ALE-161 aplicou ao tabuleiro
    // e a ALE-171 à sessão, e a mesma frase do CLAUDE.md do front: uma cena
    // preenche o espaço que recebe.
    // O painel ENCOLHE mas não some, também como na sessão: o convite é o
    // que explica para que serve aquele lado, e apagá-lo deixaria o passo
    // sem dizer o que vem depois do clique.
    <div
      class={cn(
        'grid gap-4 lg:min-h-0 lg:flex-1',
        props.value.length === 0 ? 'lg:grid-cols-[3fr_1fr]' : 'lg:grid-cols-[1.15fr_1fr]',
      )}
    >
      <div class="flex flex-col gap-3 lg:min-h-0">
        <div class="relative shrink-0">
          <Search
            class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <Input
            type="search"
            value={query()}
            onInput={(e) => setQuery(e.currentTarget.value)}
            placeholder="Buscar raça"
            class="pl-8"
            aria-label="Buscar raça"
          />
        </div>
        {/* One scroll on phone (the step column already scrolls); two panes
            with their own scroll only where they sit side by side. */}
        <div class="space-y-3 p-1 lg:min-h-0 lg:flex-1 lg:overflow-y-auto">
          <RaceTierGrid
            label="Comuns"
            names={tiers().comuns}
            value={props.value}
            onToggle={toggle}
          />
          <RaceTierGrid
            label="Outras"
            names={tiers().extras}
            value={props.value}
            onToggle={toggle}
          />
          <Show when={tiers().comuns.length === 0 && tiers().extras.length === 0}>
            <p class="p-4 text-center text-xs text-muted-foreground">
              Nenhuma raça para “{query()}”.
            </p>
          </Show>
        </div>
      </div>

      <section aria-label="Raça escolhida" class="space-y-2 lg:min-h-0 lg:overflow-y-auto lg:pr-1">
        <Show
          when={props.value.length > 0}
          fallback={
            <p class="rounded-md border border-dashed border-grimorio-iron p-4 text-center text-xs text-muted-foreground">
              Escolha uma raça para ver o que ela concede.
            </p>
          }
        >
          <For each={props.value}>
            {(name, i) => (
              <SelectedRaceDetail
                name={name}
                choice={props.choices[name] ?? {}}
                onChoice={(choice) => props.onChoice(name, choice)}
                singleRace={props.value.length === 1}
                isPrimary={i() === 0}
              />
            )}
          </For>
        </Show>
      </section>
    </div>
  )
}

function RaceTierGrid(props: {
  label: string
  names: string[]
  value: string[]
  onToggle: (name: string) => void
}) {
  return (
    <Show when={props.names.length > 0}>
      <div class="space-y-1.5">
        <SectionLabel>
          {props.label}
        </SectionLabel>
        <div
          role="listbox"
          aria-label={props.label}
          aria-multiselectable="true"
          // Column count tracks the space the grid actually gets, which is NOT
          // monotonic in viewport width: below lg the catalogue spans the whole
          // stage, at lg it gives half of it to the detail pane and needs fewer.
          class="grid grid-cols-4 gap-2 sm:grid-cols-6 md:grid-cols-8 lg:grid-cols-6 xl:grid-cols-8"
        >
          <For each={props.names}>
            {(name) => (
              <RaceTile
                name={name}
                selected={props.value.includes(name)}
                onToggle={() => props.onToggle(name)}
              />
            )}
          </For>
        </div>
      </div>
    </Show>
  )
}

function RaceTile(props: { name: string; selected: boolean; onToggle: () => void }) {
  const hue = () => hueFromName(props.name)
  return (
    <button
      type="button"
      role="option"
      aria-selected={props.selected}
      onClick={() => props.onToggle()}
      style={props.selected ? { 'outline-color': `oklch(0.6 0.16 ${hue()})` } : undefined}
      class={cn(
        'flex flex-col items-center gap-1 rounded-md border border-grimorio-iron p-1.5 transition-colors',
        props.selected ? 'bg-accent outline outline-2 outline-offset-2' : 'hover:bg-accent',
      )}
    >
      <CharacterPortrait name={props.name} size="lg" hue={hue()} class="aspect-square text-2xl" />
      <span class="w-full truncate text-center text-2xs font-medium">{props.name}</span>
      <span class="font-mono text-3xs text-muted-foreground">
        {raceSignature(props.name)}
      </span>
    </button>
  )
}

function SelectedRaceDetail(props: {
  name: string
  choice: RaceChoice
  onChoice: (next: RaceChoice) => void
  singleRace: boolean
  isPrimary: boolean
}) {
  const grant = () => raceGrant(props.name)
  /** Only an applied race contributes mechanics — the primary always does. */
  const active = () => props.isPrimary || props.choice.applied === true

  return (
    <GrantBox title={props.isPrimary ? `${props.name} · primária` : props.name}>
      <Show when={!props.isPrimary}>
        <button
          type="button"
          aria-pressed={active()}
          onClick={() => props.onChoice({ ...props.choice, applied: !props.choice.applied })}
          class={cn(
            'flex w-full items-center gap-2 rounded-sm border px-2 py-1.5 text-left text-2xs transition-colors',
            active()
              ? 'border-grimorio-gold bg-accent'
              : 'border-grimorio-iron text-muted-foreground hover:bg-accent',
          )}
        >
          <span class="flex size-3.5 shrink-0 items-center justify-center rounded-none border border-grimorio-iron">
            <Show when={active()}>
              <span class="size-2 rounded-none bg-grimorio-gold" />
            </Show>
          </span>
          Aplicar propriedades (negociado com o mestre)
        </button>
      </Show>

      <DeltaBadges deltas={resolveRaceDeltas(props.name, props.choice)} />

      <Show when={active()}>
        <RaceChoiceControls
          raceName={props.name}
          choice={props.choice}
          onChange={props.onChoice}
        />
        <DeformidadeControls
          raceName={props.name}
          choice={props.choice}
          onChange={props.onChoice}
        />
        <Show when={racePending(props.name, props.choice)}>
          <p class="text-2xs text-[color:var(--hp-hurt)]">Escolha de atributo pendente.</p>
        </Show>
      </Show>

      <Show when={grant()}>
        {(g) => (
          <AbilityDisclosure
            label="habilidades"
            singular="habilidade"
            lines={g().abilities}
            defaultOpen={props.singleRace}
          />
        )}
      </Show>
    </GrantBox>
  )
}
