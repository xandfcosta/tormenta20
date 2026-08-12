import { Sparkles, Zap } from 'lucide-solid'
import { createMemo } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { StatBox } from './stat-box'
import { pmCostRows, pmLimitRows, spellDcRows } from './stat-rows'
import type { StatsProps } from './combat-stats'

/**
 * The caster triple: how much PM one spell may take, the CD of its saves, and
 * whatever discounts the cost. Rendered only for characters who can actually
 * cast (see `ContextualStatBlocks`).
 */
export function MagicStats(props: StatsProps) {
  const sheet = createMemo(() => computedSheetFor(props.character, props.activeConditionals))

  return (
    <div class="grid grid-cols-3 gap-2">
      <StatBox
        label="Limite PM"
        dialogTitle="Limite de PM por magia"
        value={sheet().pmLimit.total}
        rows={pmLimitRows(sheet())}
        icon={Zap}
        tone="magic"
      />
      <StatBox
        label="CD Magia"
        dialogTitle="CD dos testes de resistência das suas magias"
        value={(sheet().bestBaseSpellCd ?? 0) + sheet().spellDCBonus.total}
        rows={spellDcRows(sheet())}
        icon={Sparkles}
        tone="magic"
      />
      <StatBox
        label="Custo PM"
        dialogTitle="Modificador de custo de PM"
        value={sheet().pmCostMod.total}
        rows={pmCostRows(sheet())}
        icon={Sparkles}
        tone="magic"
        signedValue
      />
    </div>
  )
}
