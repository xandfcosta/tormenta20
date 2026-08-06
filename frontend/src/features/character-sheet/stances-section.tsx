import { Flame } from 'lucide-react'
import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import {
  useAllConditionals,
  type ConditionalEntry,
} from '@/entities/character/derived'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import { usePowerAction } from '@/entities/character/use-power-action'
import { useStanceActivation } from '@/shared/stores/stance-activation-store'
import { accentStrong, subtleText, surface } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { useState } from 'react'
import { resolveConditionalDisplay } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  abbreviateConditionalTarget,
  describeConditionalTarget,
} from './conditional-target-label'
import { signed } from './signed'

type StanceGroup = { flag: string; entries: ConditionalEntry[] }

/**
 * Power-stance flag groups (FLAG_ACTIVATIONS) that are fully ON. The
 * on-switch moved to the Poderes tab; here only ACTIVE stances render.
 */
function activeStanceGroups(entries: ConditionalEntry[]): StanceGroup[] {
  const byFlag = new Map<string, ConditionalEntry[]>()
  for (const e of entries) {
    const flag = e.effect.flag
    if (!flag || !FLAG_ACTIVATIONS[flag]) continue
    byFlag.set(flag, [...(byFlag.get(flag) ?? []), e])
  }
  return [...byFlag]
    .filter(([, list]) => list.every((e) => e.active))
    .map(([flag, list]) => ({ flag, entries: list }))
}

/**
 * "Posturas ativas" — one card per running stance: what it grants, what was
 * paid (stance-activation-store, incl. the display-only stepper extra) and
 * an Encerrar exit. Hidden entirely while no stance is up.
 */
export function StancesSection({ character }: { character: Character }) {
  const entries = useAllConditionals(character)
  const groups = activeStanceGroups(entries)
  if (groups.length === 0) return null
  return (
    <section className={cn('rounded-lg border p-3', surface)}>
      <h3 className={cn('text-sm font-bold', accentStrong)}>Posturas ativas</h3>
      <ul className="mt-2 space-y-1">
        {groups.map((g) => (
          <StanceCard key={g.flag} group={g} character={character} />
        ))}
      </ul>
    </section>
  )
}

/**
 * One-line stance summary (owner feedback: the old card listed 9 rows incl.
 * the superseded Fúria +2 tier). Runs the ENGINE's non-stacking resolution
 * over the flag group so only winning tiers render, then groups surviving
 * targets by amount: "Atq/Dano/Fort/Von +3". Tapping the line (not Encerrar)
 * expands the audited per-target breakdown.
 */
function StanceCard({
  group,
  character,
}: {
  group: StanceGroup
  character: Character
}) {
  const { deactivateStance } = usePowerAction(character)
  const [expanded, setExpanded] = useState(false)
  const sheet = useComputedSheet(character)
  const activation = FLAG_ACTIVATIONS[group.flag]
  // Fallback to the base cost for stances toggled before this phase (no record).
  const paid = useStanceActivation(character.id, group.flag)
  const pmPaid = paid?.pmPaid ?? activation.pmCost
  const tempHp = group.flag === 'furia' ? sheet.tempHpFuria : null
  const summary = stanceSummary(group)
  return (
    <li className="rounded-md border border-violet-500/30 bg-violet-50/60 px-2 py-1.5 dark:border-violet-500/25 dark:bg-violet-950/30">
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          className="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-left"
        >
          <Flame className="size-3.5 shrink-0 text-violet-700 dark:text-violet-300" />
          <span className="shrink-0 text-sm font-medium text-foreground">
            {activation.name}
          </span>
          <span className={cn('min-w-0 truncate text-[11px]', subtleText)}>
            · {summary}
            {tempHp && tempHp.total > 0 ? ` · PV temp +${tempHp.total}` : ''}
            {` · ${pmPaid} PM`}
          </span>
        </button>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-11 shrink-0 px-3 text-xs text-red-700 hover:bg-red-100 hover:text-red-700 dark:text-red-400 dark:hover:bg-red-950/40 sm:h-6 sm:px-2 sm:text-[11px]"
          onClick={() => deactivateStance(group.flag)}
          aria-label={`Encerrar ${activation.name}`}
        >
          Encerrar
        </Button>
      </div>
      {expanded && <StanceBreakdown group={group} character={character} />}
    </li>
  )
}

/** "Atq/Dano/Fort/Von +3" — winning tiers only, grouped by signed amount. */
function stanceSummary(group: StanceGroup): string {
  const kept = resolveConditionalDisplay(
    group.entries.map((e) => ({
      target: e.effect.target,
      bonusType: e.effect.bonusType,
      amount: e.effect.amount,
    })),
  )
  const byAmount = new Map<number, string[]>()
  for (const k of kept) {
    const list = byAmount.get(k.amount) ?? []
    list.push(abbreviateConditionalTarget(k.target))
    byAmount.set(k.amount, list)
  }
  return [...byAmount]
    .map(([amount, labels]) => `${labels.join('/')} ${signed(amount)}`)
    .join(' · ')
}

/** Per-modifier lines (FlagGroupRow's inner-list markup) + temp-PV when owned. */
function StanceBreakdown({
  group,
  character,
}: {
  group: StanceGroup
  character: Character
}) {
  const sheet = useComputedSheet(character)
  const tempHp = group.flag === 'furia' ? sheet.tempHpFuria : null
  return (
    <ul className="ml-5 mt-1 space-y-0.5 text-[11px]">
      {group.entries.map((e) => (
        <li key={e.id} className="flex items-center justify-between gap-2">
          <span className={cn('truncate', subtleText)}>
            {describeConditionalTarget(e.effect.target)}
          </span>
          <span
            className={cn(
              'shrink-0 font-mono font-semibold',
              e.effect.amount >= 0
                ? 'text-emerald-700 dark:text-emerald-300'
                : 'text-red-700 dark:text-red-300',
            )}
          >
            {signed(e.effect.amount)}
          </span>
        </li>
      ))}
      {tempHp && tempHp.total > 0 && (
        <li className="flex items-center justify-between gap-2">
          <span className={cn('truncate', subtleText)}>
            Alma de Bronze — PV temporários (nível + For)
          </span>
          <span className="shrink-0 font-mono font-semibold text-emerald-700 dark:text-emerald-300">
            +{tempHp.total}
          </span>
        </li>
      )}
    </ul>
  )
}
