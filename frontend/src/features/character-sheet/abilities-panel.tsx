import { getRace } from '@tormenta20/t20-data'
import type { RaceDefinition } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { AbilitiesSection } from './abilities-section'
import { ClassesSection } from './class-abilities'
import { OriginAbilitySection } from './origin-abilities'
import { RaceAbilitySection } from './race-abilities'

/**
 * "Habilidades" tab — aggregates race, origin, and class ability
 * sub-sections. Each character can have multiple races (via
 * `character.races[]`) so the section renders one race block per row.
 * Class sections render similarly per multiclass entry.
 */
export function AbilitiesPanel({ character }: { character: Character }) {
  const races = character.races
    .map((r) => getRace(r.race))
    .filter((r): r is RaceDefinition => Boolean(r))
  const classes = character.classes
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      {races.length === 0 ? (
        <AbilitiesSection title={`Raça: —`}>
          <p className={cn('text-xs italic', dimText)}>
            Raça do personagem não está no catálogo.
          </p>
        </AbilitiesSection>
      ) : (
        races.map((race) => (
          <RaceAbilitySection
            key={race.id}
            race={race}
            character={character}
          />
        ))
      )}
      <OriginAbilitySection character={character} />
      {classes.length === 0 ? (
        <AbilitiesSection title="Classes: —">
          <p className={cn('text-xs italic', dimText)}>
            Nenhuma classe atribuída.
          </p>
        </AbilitiesSection>
      ) : (
        classes.map((entry) => (
          <ClassesSection
            key={entry.className}
            entry={entry}
            character={character}
          />
        ))
      )}
    </div>
  )
}
