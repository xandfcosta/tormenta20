import { ConditionPips } from './conditions-section'
import { Minus, Plus } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { cn } from '@/shared/lib/utils'
import type { Character } from '@/shared/api/api'
import { LevelBadge, SheetIdentityText } from './sheet-header'
import { CombatStats, MagicStats } from './combat-magic-stats'
import { AttributesGrid } from './vitals-aside'
import { ResourceAdjustDialog } from './resource-bar'
import { useVitals } from './use-vitals'

/**
 * Game-style HUD pinned to the bottom of the sheet on both layouts. A player
 * "nameplate" on the left — square portrait beside the name/subtitle stacked
 * over the PV/PM bars they control — with the class badges + level stepper on
 * the plate, and (on desktop) the attribute/combat/magic stat blocks filling
 * the rest of the bar. Replaces the old top header.
 */
export function CharacterHud({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  const { setHp, setMp } = useVitals(character)
  return (
    <div className={cn('border-t bg-card px-3 py-2 sm:px-4', className)}>
      <div className="flex items-stretch gap-3 lg:gap-4">
        {/* Nameplate: square portrait + [name/info over PV/PM bars]. Fills the
            width on phones; a fixed slice on desktop so the stats get the rest. */}
        <div className="flex min-w-0 flex-1 items-stretch gap-3 lg:w-[34rem] lg:flex-none">
          <CharacterPortrait
            name={character.name}
            size="sm"
            className="aspect-square h-auto w-auto self-stretch rounded-xl text-2xl"
          />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex items-start justify-between gap-2">
              <SheetIdentityText character={character} />
              <div className="shrink-0">
                <LevelBadge character={character} />
              </div>
            </div>
            {/* Class badges sit under the info, just above the PV/PM bars. */}
            <ClassBadges character={character} className="flex-wrap" />
            <ConditionPips character={character} />
            <div className="mt-auto flex flex-col gap-1">
              <HudVital
                label="Vida"
                current={character.hpCurrent}
                max={character.hpMax}
                kind="hp"
                onSet={setHp}
              />
              <HudVital
                label="Mana"
                current={character.mpCurrent}
                max={character.mpMax}
                kind="mp"
                onSet={setMp}
              />
            </div>
          </div>
        </div>

        {/* Desktop only: two short rows that expand to fill the rest of the bar
            — attributes on top, combat + magic below. The wide region keeps
            every box's corner icon clear of its label. */}
        <div className="hidden min-w-0 flex-1 flex-col justify-center gap-1.5 lg:flex">
          <AttributesGrid character={character} className="grid-cols-6" />
          {/* Side by side only when wide enough; stacked below `xl` so the
              combat/magic boxes stay wide (label never crowds the icon). */}
          <div className="grid grid-cols-1 gap-2 xl:grid-cols-2">
            <CombatStats character={character} />
            <MagicStats character={character} />
          </div>
        </div>
      </div>
    </div>
  )
}

/** The character's per-class level badges (e.g. "Bardo 3"). */
function ClassBadges({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  return (
    <div className={cn('flex gap-1', className)}>
      {character.classes.map((c) => (
        <Badge
          key={c.className}
          className="px-1.5 py-0 text-[10px] leading-tight sm:px-2 sm:py-0.5 sm:text-xs"
        >
          {c.className} {c.level}
        </Badge>
      ))}
    </div>
  )
}

/** HP fill token by ratio — the color, not just width, signals "how bad". */
function hpFillVar(pct: number): string {
  if (pct <= 25) return '--hp-critical'
  if (pct <= 50) return '--hp-hurt'
  return '--hp-full'
}

/**
 * One compact PV/PM row: label · decay-colored fill bar · current/max · −/+ ·
 * bulk-edit. Fill is driven by the semantic hp/mp tokens (HP shifts
 * green→amber→red as it drops) with a real `progressbar` role. Steppers are
 * finger-sized on phones and compact from `lg` up.
 */
function HudVital({
  label,
  current,
  max,
  kind,
  onSet,
}: {
  label: string
  current: number
  max: number
  kind: 'hp' | 'mp'
  onSet: (next: number) => void
}) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  const fillVar = kind === 'hp' ? hpFillVar(pct) : '--mp-arcane'
  return (
    <div className="flex items-center gap-2">
      <span
        className="w-9 shrink-0 text-[10px] font-bold uppercase tracking-wider"
        style={{ color: `var(${fillVar})` }}
      >
        {label}
      </span>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuenow={current}
        aria-valuemin={0}
        aria-valuemax={max}
        className="relative h-2.5 flex-1 overflow-hidden rounded-full border border-border bg-muted"
      >
        <div
          className="h-full transition-[width,background-color] duration-500 ease-out"
          style={{ width: `${pct}%`, backgroundColor: `var(${fillVar})` }}
        />
      </div>
      <span className="shrink-0 font-mono text-xs tabular-nums">
        <span className="font-bold">{current}</span>
        <span className="text-muted-foreground">/{max}</span>
      </span>
      <div className="flex shrink-0 items-center gap-0.5">
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-9 lg:size-6"
          disabled={current <= 0}
          onClick={() => onSet(current - 1)}
          aria-label={`Reduzir ${label} em 1`}
        >
          <Minus className="size-4 lg:size-3" />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-9 lg:size-6"
          disabled={current >= max}
          onClick={() => onSet(current + 1)}
          aria-label={`Aumentar ${label} em 1`}
        >
          <Plus className="size-4 lg:size-3" />
        </Button>
        <ResourceAdjustDialog
          label={label}
          current={current}
          max={max}
          onSetCurrent={onSet}
          triggerClassName="size-9 lg:size-6"
        />
      </div>
    </div>
  )
}
