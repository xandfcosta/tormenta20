import { memo, useEffect, useRef, useState } from 'react'
import { ConditionPips } from './conditions-section'
import { MobileDefChip } from './mobile-def-chip'
import { Minus, Plus } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { cn } from '@/shared/lib/utils'
import { tempHpPool } from '@/entities/character/temp-hp-pool'
import { useApplyDamage } from '@/entities/character/use-apply-damage'
import { useManualTempHp } from '@/entities/character/use-manual-temp-hp'
import type { Character } from '@/shared/api/api'
import { LevelBadge, SheetIdentityText } from './sheet-header'
import { CombatStats, SavesStats } from './combat-magic-stats'
import { AttributesGrid, ContextualStatBlocks } from './vitals-aside'
import { ResourceAdjustDialog } from './resource-bar'
import { useVitals } from './use-vitals'

/**
 * Game-style HUD pinned to the bottom of the sheet on both layouts. A player
 * "nameplate" on the left — square portrait beside the name/subtitle stacked
 * over the PV/PM bars they control — with the class badges + level stepper on
 * the plate, and (on desktop) the attribute/combat/magic stat blocks filling
 * the rest of the bar. Replaces the old top header.
 */
// Memoized: the HUD is the sheet's heaviest always-mounted chrome (portrait,
// PV/PM bars, the full attribute/combat/magic tile grid via combat-magic-stats
// with 3 useComputedSheet reads). A tab switch re-renders the layout but not
// the HUD's `character`/`className`, so memo lets it bail out of that churn —
// this is what collapses the ~160ms per-switch floor left after the panels
// were memoized. HP/PM edits mint a new character object, so it still updates.
function CharacterHudImpl({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  const { setHp, setMp } = useVitals(character)
  // Fase 4: the REAL pool (persisted tempHp effects) replaced the display-only
  // tempHpFromPowers here. F2: damage goes through ONE atomic POST :id/damage
  // (server drains the pool first, remainder lowers hp) — the old two-write
  // drain path is gone. Healing keeps the optimistic vitals PATCH.
  const pool = tempHpPool(character)
  const { applyDamage } = useApplyDamage(character)
  const { setManualTempHp } = useManualTempHp(character)
  const applyHp = (next: number) => {
    const damage = character.hpCurrent - next
    if (damage <= 0) {
      setHp(next)
      return
    }
    applyDamage(damage)
  }
  return (
    <div className={cn('border-t bg-card px-3 py-2 sm:px-4', className)}>
      {/* md-only stacks the stat cluster BELOW the nameplate: side-by-side the
          cluster's height stretched the square portrait, which squeezed the
          PV/PM row until its +/−/edit buttons slid under the cluster and were
          untappable (audit task 10). From lg the desktop row layout returns. */}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-stretch lg:gap-4">
        {/* Nameplate: square portrait + [name/info over PV/PM bars]. Fills the
            width below lg; a fixed slice on desktop so the stats get the rest. */}
        <div className="flex min-w-0 items-stretch gap-3 lg:w-[34rem] lg:flex-none">
          {/* Fixed-width card that fills the nameplate height (h-auto lets the flex
              row stretch it). `aspect-square + self-stretch` was buggy: aspect-ratio
              can't derive width from a flex-stretched height, so it collapsed to the
              initials' width — a thin clipped strip. */}
          <CharacterPortrait
            name={character.name}
            size="sm"
            className="h-auto w-24 self-stretch rounded-md border-2 border-grimorio-iron text-4xl"
          />
          <div className="flex min-w-0 flex-1 flex-col gap-1.5">
            <div className="flex items-start justify-between gap-2">
              <SheetIdentityText character={character} />
              <div className="flex shrink-0 items-center gap-1.5">
                <MobileDefChip character={character} className="md:hidden" />
                <LevelBadge character={character} />
              </div>
            </div>
            {/* Class badges sit under the info, just above the PV/PM bars. */}
            {/* Badges + condition mini-pips share ONE row — a dedicated
                conditions row doubled the nameplate height (owner feedback). */}
            <div className="flex flex-wrap items-center gap-1">
              <ClassBadges character={character} />
              <ConditionPips character={character} mini />
            </div>
            <div className="mt-auto flex flex-col gap-1">
              <HudVital
                label="Vida"
                current={character.hpCurrent}
                max={character.hpMax}
                kind="hp"
                onSet={applyHp}
                onDamage={applyDamage}
                temp={pool.total}
                tempTitle={pool.slices.map((s) => s.label).join(', ')}
                tempPool={{ total: pool.total, onSetManual: setManualTempHp }}
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


        {/* From md (tablet): the stat cluster — this IS the Vitais content, so
            the mobile layout drops its Vitais tab at these widths (task 12).
            md gets full-width 2-col rows under the nameplate; lg stacks 1-col
            beside it; xl spreads back to 2-col. */}
        <div className="hidden min-w-0 flex-1 flex-col justify-center gap-1.5 md:flex">
          {/* Row A: the reactive numbers — defense/attacks + the three saves
              ("teste de Reflexos!"). Row B: contextual — weapon formulas when
              something is wielded AND the magic triple for casters (hybrids
              get both); attributes keep a compact provenance row. */}
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            <CombatStats character={character} />
            <SavesStats character={character} />
          </div>
          <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            <AttributesGrid character={character} className="grid-cols-6" />
            <ContextualStatBlocks character={character} />
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
  onDamage,
  temp = 0,
  tempTitle,
  tempPool,
}: {
  label: string
  current: number
  max: number
  kind: 'hp' | 'mp'
  onSet: (next: number) => void
  /** F2: routes "Remover" through POST :id/damage (temp-first, atomic). */
  onDamage?: (amount: number) => void
  /** Debitable temp-PV pool (persisted tempHp effects) shown as "+N". */
  temp?: number
  tempTitle?: string
  /** F3: pool-aware ✎ dialog — current total + manual pool setter. */
  tempPool?: { total: number; onSetManual: (value: number) => void }
}) {
  const pct = max > 0 ? Math.max(0, Math.min(100, (current / max) * 100)) : 0
  const fillVar = kind === 'hp' ? hpFillVar(pct) : '--mp-arcane'
  const delta = useVitalDelta(current)
  // Shift-click steps ±5 — combat deltas are rarely 1.
  const stepOf = (e: React.MouseEvent) => (e.shiftKey ? 5 : 1)
  return (
    <div className="flex items-center gap-1.5 sm:gap-2">
      <span
        className="w-9 shrink-0 text-[10px] font-bold uppercase tracking-wider"
        style={{ color: `var(${fillVar})` }}
      >
        {label}
      </span>
      {/* − on the far side of + so a greasy thumb never heals when it meant
          to hurt (audit: fat-finger risk at gap-0.5). Damage goes through the
          atomic endpoint UNclamped (F2) — the server routes temp-first, so a
          shift−5 at 3 PV still drains 5 from the pool; with a pool left the
          button stays tappable even at 0 PV. */}
      <Button
        type="button"
        variant="outline"
        size="icon"
        className="size-9 shrink-0 lg:size-6"
        disabled={current <= 0 && temp <= 0}
        onClick={(e) =>
          onDamage
            ? onDamage(stepOf(e))
            : onSet(Math.max(0, current - stepOf(e)))
        }
        aria-label={`Reduzir ${label} (shift: 5)`}
      >
        <Minus className="size-4 lg:size-3" />
      </Button>
      <div
        role="progressbar"
        aria-label={label}
        aria-valuenow={current}
        aria-valuemin={0}
        aria-valuemax={max}
        className="relative h-3.5 min-w-8 flex-1 overflow-hidden rounded-full border border-border bg-muted lg:h-2.5"
      >
        <div
          className="h-full transition-[width,background-color] duration-500 ease-out"
          style={{ width: `${pct}%`, backgroundColor: `var(${fillVar})` }}
        />
      </div>
      <span className="relative shrink-0 font-mono text-base tabular-nums lg:text-xs">
        <span className="font-bold">{current}</span>
        <span className="text-muted-foreground">/{max}</span>
        {temp > 0 && (
          <span
            className="ml-1 font-bold text-emerald-700 dark:text-emerald-400"
            title={tempTitle ?? 'PV temporários'}
          >
            +{temp}
          </span>
        )}
        {delta !== null && (
          <span
            className={cn(
              'absolute -top-4 right-0 animate-in fade-in-0 slide-in-from-bottom-1 text-[10px] font-bold',
              delta < 0
                ? 'text-[color:var(--hp-critical)]'
                : 'text-[color:var(--hp-full)]',
            )}
          >
            {delta > 0 ? `+${delta}` : delta}
          </span>
        )}
      </span>
      <div className="flex shrink-0 items-center gap-1 lg:gap-0.5">
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="size-9 lg:size-6"
          disabled={current >= max}
          onClick={(e) => onSet(Math.min(max, current + stepOf(e)))}
          aria-label={`Aumentar ${label} (shift: 5)`}
        >
          <Plus className="size-4 lg:size-3" />
        </Button>
        <ResourceAdjustDialog
          label={label}
          current={current}
          max={max}
          onSetCurrent={onSet}
          onDamage={onDamage}
          tempPool={tempPool}
          triggerClassName="size-9 lg:size-6"
        />
      </div>
    </div>
  )
}

/**
 * Lingering delta chip: any change to `current` (own click OR external — GM
 * socket damage, expired effects) shows "+X/−X" for a few seconds, so a
 * player looking away still sees WHAT changed, not just the new number.
 */
function useVitalDelta(current: number): number | null {
  const prevRef = useRef(current)
  const [delta, setDelta] = useState<number | null>(null)
  useEffect(() => {
    const diff = current - prevRef.current
    prevRef.current = current
    if (diff === 0) return
    setDelta(diff)
    const timer = setTimeout(() => setDelta(null), 3000)
    return () => clearTimeout(timer)
  }, [current])
  return delta
}

export const CharacterHud = memo(CharacterHudImpl)
