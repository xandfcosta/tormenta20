import type { Character } from '@/shared/api/api'
import { attributeTotal, useCharacterEffects } from '@/entities/character/derived'
import { cn } from '@/shared/lib/utils'
import { CombatStats, MagicStats } from './combat-magic-stats'

// PV/PM live in the bottom HUD (see `CharacterHud`); this column now holds the
// static vitals — attributes and the combat/magic stat blocks.
export function VitalsAside({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  return (
    <aside
      className={cn(
        'flex min-h-0 flex-col gap-3 overflow-y-auto rounded-xl border bg-card p-3 sm:p-4',
        className,
      )}
    >
      <AttributesGrid
        character={character}
        className="grid-cols-3 sm:grid-cols-6 lg:grid-cols-3"
      />

      <CombatStats character={character} />
      <MagicStats character={character} />
    </aside>
  )
}

/** The six attribute boxes. `className` sets the column template so callers
 *  (aside column vs. HUD) control how they wrap. */
export function AttributesGrid({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  // Stored attributes are BASE (pre-race); the racial mod is folded via the
  // character effects (race active items), so show the derived total.
  const effects = useCharacterEffects(character)
  return (
    <div className={cn('grid gap-2', className)}>
      <AttributeBox label="FOR" value={attributeTotal(character, 'strength', effects)} />
      <AttributeBox label="DES" value={attributeTotal(character, 'dexterity', effects)} />
      <AttributeBox label="CON" value={attributeTotal(character, 'constitution', effects)} />
      <AttributeBox label="INT" value={attributeTotal(character, 'intelligence', effects)} />
      <AttributeBox label="SAB" value={attributeTotal(character, 'wisdom', effects)} />
      <AttributeBox label="CAR" value={attributeTotal(character, 'charisma', effects)} />
    </div>
  )
}

function AttributeBox({ label, value }: { label: string; value: number }) {
  const sign = value >= 0 ? '+' : ''
  return (
    <div className="rounded-lg border-2 p-2 text-center">
      <p className="text-[9px] font-bold uppercase tracking-widest text-muted-foreground">
        {label}
      </p>
      <p className="mt-0.5 text-2xl font-bold leading-none text-foreground">
        {sign}
        {value}
      </p>
    </div>
  )
}
