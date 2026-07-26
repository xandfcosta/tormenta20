import type { Character } from '@/shared/api/api'
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
  return (
    <div className={cn('grid gap-2', className)}>
      <AttributeBox label="FOR" value={character.strength} />
      <AttributeBox label="DES" value={character.dexterity} />
      <AttributeBox label="CON" value={character.constitution} />
      <AttributeBox label="INT" value={character.intelligence} />
      <AttributeBox label="SAB" value={character.wisdom} />
      <AttributeBox label="CAR" value={character.charisma} />
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
