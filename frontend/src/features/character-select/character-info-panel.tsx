import { Link } from '@tanstack/react-router'
import { useMemo } from 'react'
import type { Character } from '@/shared/api/api'
import { AbilityLine } from '@/shared/ui/ability-line'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { StatCell } from '@/shared/ui/stat-cell'
import { hueFromName } from '@/shared/lib/hue-from-name'
import {
  characterEffects,
  defenseTotal,
} from '@/entities/character/derived'
import {
  type AbilityBlurb,
  characterFlavor,
  primaryRole,
  raceAbilityBlurbs,
  raceLoreLine,
} from './select-helpers'

/**
 * Right-side info panel of the character-select screen. Distributes the T20
 * data into Valorant-style slots: role tag, name, a DEF·PV·PM stat triple,
 * an assembled flavor paragraph, and a race-ability blurb row. Defesa is the
 * one derived number — computed here for the selected character only (never
 * per roster thumbnail) since it runs the full effects engine.
 */
export function CharacterInfoPanel({ character }: { character: Character }) {
  const defense = useMemo(
    () => defenseTotal(character, characterEffects(character)).total,
    [character],
  )
  const hue = hueFromName(character.name)
  const flavor = characterFlavor(character)
  const lore = raceLoreLine(character)
  const abilities = raceAbilityBlurbs(character, 4)

  return (
    <Card className="min-h-0 min-w-0 gap-0 self-start overflow-y-auto py-4 lg:max-h-full lg:py-6">
      <CardContent className="flex flex-col gap-4 px-4 lg:px-6">
        <div>
          <p className="text-xs font-semibold uppercase tracking-widest text-muted-foreground">
            {primaryRole(character)}
          </p>
          <h2 className="text-2xl font-semibold tracking-tight">
            {character.name}
          </h2>
        </div>

        <StatTriple defense={defense} hue={hue} character={character} />

        <div className="space-y-1">
          <p className="text-sm text-muted-foreground">{flavor}</p>
          {lore && (
            <p className="text-xs italic leading-snug text-muted-foreground/80">
              {lore}
            </p>
          )}
        </div>

        <AbilityList abilities={abilities} />

        <Link to="/characters/$id" params={{ id: character.id }}>
          <Button className="w-full" size="lg">
            Abrir ficha
          </Button>
        </Link>
      </CardContent>
    </Card>
  )
}

/**
 * Compact one-row summary of the selected character. Shown above the expanded
 * roster grid so scanning a big roster doesn't mean picking blind — the user
 * still sees who's selected + the DEF·PV·PM triple + the open action.
 */
export function CharacterSummaryBar({ character }: { character: Character }) {
  const defense = useMemo(
    () => defenseTotal(character, characterEffects(character)).total,
    [character],
  )
  const hue = hueFromName(character.name)
  return (
    <Card className="min-w-0 shrink-0 flex-row flex-wrap items-center gap-3 px-4 py-3">
      <div className="min-w-0">
        <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
          {primaryRole(character)}
        </p>
        <h2 className="truncate text-lg font-semibold tracking-tight">
          {character.name}
        </h2>
      </div>
      <div className="ml-auto flex items-center gap-3 font-mono text-sm tabular-nums">
        <span style={{ color: `oklch(0.62 0.16 ${hue})` }}>DEF {defense}</span>
        <span>
          PV {character.hpCurrent}/{character.hpMax}
        </span>
        <span>
          PM {character.mpCurrent}/{character.mpMax}
        </span>
      </div>
      <Link to="/characters/$id" params={{ id: character.id }}>
        <Button size="sm">Abrir ficha</Button>
      </Link>
    </Card>
  )
}

/** Fill token for a current/max ratio — mirrors the HpBar decay states. */
function hpToken(current: number, max: number): string {
  const ratio = max > 0 ? current / max : 0
  if (ratio <= 0.25) return '--hp-critical'
  if (ratio <= 0.5) return '--hp-hurt'
  return '--hp-full'
}

/**
 * DEF · PV · PM — the at-a-glance combat triple. DEF is the single derived
 * number (tinted with the character's hue as the panel's second focal point);
 * PV/PM show current/max because at a live table the current value is what a
 * player actually checks. PV is colored by its decay ratio; a 0-max PM cell is
 * dimmed (martials).
 */
function StatTriple({
  defense,
  hue,
  character,
}: {
  defense: number
  hue: number
  character: Character
}) {
  const noPm = character.mpMax === 0
  return (
    <div className="grid grid-cols-3 gap-2">
      <StatCell label="DEF">
        <span style={{ color: `oklch(0.62 0.16 ${hue})` }}>{defense}</span>
      </StatCell>
      <StatCell label="PV">
        <span style={{ color: `var(${hpToken(character.hpCurrent, character.hpMax)})` }}>
          {character.hpCurrent}
        </span>
        <span className="text-base text-muted-foreground">/{character.hpMax}</span>
      </StatCell>
      <StatCell label="PM" dim={noPm}>
        <span style={noPm ? undefined : { color: 'var(--mp-arcane)' }}>
          {character.mpCurrent}
        </span>
        <span className="text-base text-muted-foreground">/{character.mpMax}</span>
      </StatCell>
    </div>
  )
}

function AbilityList({ abilities }: { abilities: AbilityBlurb[] }) {
  if (abilities.length === 0) {
    return (
      <p className="text-xs italic text-muted-foreground">
        Sem habilidades de raça catalogadas.
      </p>
    )
  }
  return (
    <div className="space-y-2">
      <p className="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground">
        Habilidades
      </p>
      <ul className="space-y-1.5">
        {abilities.map((ability) => (
          <AbilityLine
            key={ability.id}
            category={ability.category ?? undefined}
            name={ability.name}
            description={ability.description}
          />
        ))}
      </ul>
    </div>
  )
}
