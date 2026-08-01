import { Search } from 'lucide-react'
import { useState } from 'react'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { Input } from '@/shared/ui/input'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import {
  type RaceChoice,
  type RaceChoiceState,
  raceGrant,
  racePending,
  racesByTier,
  raceSignature,
  resolveRaceDeltas,
} from './grant-helpers'
import { AbilityDisclosure, DeltaBadges, GrantBox } from './grant-panels'
import { RaceChoiceControls } from './race-choice-controls'

/**
 * Race picker — a searchable, tier-grouped grid of hue-tiled race tiles
 * (mirroring the roster) with the signature attribute delta on each tile.
 * Multi-select (homebrew): the FIRST picked race is the mechanical **primary**
 * (its mods apply); extra races are flavor whose benefits/debuffs are negotiated
 * with the GM (not applied automatically). Each selected race expands a detail
 * box: resolved deltas + inline floating/subrace choice capture + abilities.
 */
export function RacePicker({
  options,
  value,
  choices,
  onChange,
  onChoicesChange,
}: {
  options: string[]
  value: string[]
  choices: RaceChoiceState
  onChange: (next: string[]) => void
  onChoicesChange: (next: RaceChoiceState) => void
}) {
  const [query, setQuery] = useState('')
  const q = query.trim().toLowerCase()
  const filtered = q ? options.filter((n) => n.toLowerCase().includes(q)) : options
  const { comuns, extras } = racesByTier(filtered)

  // Multi-select: toggle a race in/out. `value[0]` stays the mechanical primary.
  const toggle = (name: string) =>
    onChange(
      value.includes(name)
        ? value.filter((n) => n !== name)
        : [...value, name],
    )
  const setChoice = (name: string, choice: RaceChoice) =>
    onChoicesChange({ ...choices, [name]: choice })

  return (
    <div className="space-y-3 lg:grid lg:grid-cols-[1.1fr_1fr] lg:gap-4 lg:space-y-0">
      {/* Left: search (fixed) + scrollable tile catalog */}
      <div className="flex flex-col gap-3">
        <div className="relative shrink-0">
          <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar raça"
            className="pl-8"
            aria-label="Buscar raça"
          />
        </div>
        <div className="max-h-[min(340px,42vh)] space-y-3 overflow-y-auto p-1.5">
          <RaceTierGrid label="Comuns" names={comuns} value={value} onToggle={toggle} />
          <RaceTierGrid label="Outras" names={extras} value={value} onToggle={toggle} />
        </div>
      </div>
      {/* Right: pinned selected-race detail */}
      <div className="space-y-2 lg:max-h-[min(440px,54vh)] lg:overflow-y-auto lg:pr-1">
        {value.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
            Selecione uma raça para ver os detalhes.
          </p>
        ) : (
          value.map((name, i) => (
            <SelectedRaceDetail
              key={name}
              name={name}
              choice={choices[name] ?? {}}
              onChoice={(c) => setChoice(name, c)}
              singleRace={value.length === 1}
              isPrimary={i === 0}
            />
          ))
        )}
      </div>
    </div>
  )
}

function RaceTierGrid({
  label,
  names,
  value,
  onToggle,
}: {
  label: string
  names: string[]
  value: string[]
  onToggle: (name: string) => void
}) {
  if (names.length === 0) return null
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold uppercase tracking-widest text-muted-foreground">
        {label}
      </p>
      <div
        role="listbox"
        aria-label={label}
        aria-multiselectable
        className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-6"
      >
        {names.map((name) => (
          <RaceTile
            key={name}
            name={name}
            selected={value.includes(name)}
            onToggle={() => onToggle(name)}
          />
        ))}
      </div>
    </div>
  )
}

function RaceTile({
  name,
  selected,
  onToggle,
}: {
  name: string
  selected: boolean
  onToggle: () => void
}) {
  const hue = hueFromName(name)
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      onClick={onToggle}
      style={selected ? { outlineColor: `oklch(0.6 0.16 ${hue})` } : undefined}
      className={cn(
        'flex flex-col items-center gap-1 rounded-lg border border-border p-1.5 transition-colors',
        selected ? 'bg-accent outline outline-2 outline-offset-2' : 'hover:bg-accent',
      )}
    >
      <CharacterPortrait
        name={name}
        size="lg"
        hue={hue}
        className="aspect-square text-2xl"
      />
      <span className="w-full truncate text-center text-[11px] font-medium">
        {name}
      </span>
      <span className="font-mono text-[10px] text-muted-foreground">
        {raceSignature(name)}
      </span>
    </button>
  )
}

function SelectedRaceDetail({
  name,
  choice,
  onChoice,
  singleRace,
  isPrimary,
}: {
  name: string
  choice: RaceChoice
  onChoice: (next: RaceChoice) => void
  singleRace: boolean
  isPrimary: boolean
}) {
  const grant = raceGrant(name)
  const active = isPrimary || choice.applied === true
  return (
    <GrantBox title={isPrimary ? `${name} · primária` : name}>
      {!isPrimary && (
        <button
          type="button"
          aria-pressed={active}
          onClick={() => onChoice({ ...choice, applied: !choice.applied })}
          className={cn(
            'flex w-full items-center gap-2 rounded-md border px-2 py-1.5 text-left text-[11px] transition-colors',
            active
              ? 'border-primary bg-accent'
              : 'border-border text-muted-foreground hover:bg-accent',
          )}
        >
          <span className="flex size-3.5 shrink-0 items-center justify-center rounded-sm border border-border">
            {active && <span className="size-2 rounded-sm bg-primary" />}
          </span>
          Aplicar propriedades (negociado com o mestre)
        </button>
      )}
      <DeltaBadges deltas={resolveRaceDeltas(name, choice)} />
      {active && (
        <RaceChoiceControls raceName={name} choice={choice} onChange={onChoice} />
      )}
      {active && racePending(name, choice) && (
        <p className="text-[11px] text-[color:var(--hp-hurt)]">
          Escolha de atributo pendente.
        </p>
      )}
      {grant && (
        <AbilityDisclosure
          label="habilidades"
          singular="habilidade"
          lines={grant.abilities}
          defaultOpen={singleRace}
        />
      )}
    </GrantBox>
  )
}
