import { Zap } from 'lucide-react'
import type {
  ActivationAction,
  ActivationSpec,
  ActivationUses,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { useAllConditionals } from '@/entities/character/derived'
import {
  usePowerAction,
  enforcedScopeOf,
  stanceActivationDecision,
  stanceFlagOf,
} from '@/entities/character/use-power-action'
import { usePowerUsedCounts } from '@/shared/stores/power-uses-store'
import { subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { UsePowerDialog } from './use-power-dialog'

const ACTION_LABEL: Record<ActivationAction, string> = {
  padrao: 'PADRÃO',
  movimento: 'MOVIMENTO',
  livre: 'LIVRE',
  reacao: 'REAÇÃO',
  gratuita: 'GRATUITA',
  completa: 'COMPLETA',
  passivo: 'PASSIVA',
  varia: 'VARIA',
}

function usesBadge(uses: ActivationUses): string | null {
  if (uses === null) return null
  if (typeof uses === 'number') return `${uses}/dia`
  return `1/${uses}`
}

const CHIP =
  'shrink-0 rounded-full border border-border px-1.5 py-px text-[9px] font-semibold uppercase tracking-wide'

/**
 * Inline use affordance for one ability row of the Poderes tab. Renders per
 * the activation taxonomy: passives get a chip, triggered passives a live
 * gatilho dot, instants the one-tap "Usar N PM" button, stances an Ativar
 * button (stepper dialog when the stance scales). `undefined` spec means
 * the ability has no activation entry — render nothing, the row stays as-is.
 */
export function PowerActionSlot({
  spec,
  character,
  className,
}: {
  spec: ActivationSpec | undefined
  character: Character
  className?: string
}) {
  if (!spec) return null
  return (
    <span className={cn('flex flex-wrap items-center gap-1', className)}>
      {spec.kind === 'passive' && <span className={cn(CHIP, subtleText)}>Passiva</span>}
      {spec.kind === 'triggered-passive' && (
        <TriggeredPassiveChip spec={spec} character={character} />
      )}
      {spec.kind === 'stance' && <StanceSlot spec={spec} character={character} />}
      {spec.kind === 'instant' && (
        <>
          <ActionCostChip spec={spec} />
          <InstantUseButton spec={spec} character={character} />
        </>
      )}
    </span>
  )
}


/** `LIVRE · 1 PM`-style chip: what the use costs in action economy + PM. */
function ActionCostChip({ spec }: { spec: ActivationSpec }) {
  const pm = spec.pmCost === 'variavel' ? 'PM variável' : `${spec.pmCost} PM`
  return (
    <span className={cn(CHIP, subtleText)}>
      {ACTION_LABEL[spec.action]} · {pm}
    </span>
  )
}

/** Passive that only fires while its stance flag is up — live ●/○ dot. */
function TriggeredPassiveChip({
  spec,
  character,
}: {
  spec: ActivationSpec
  character: Character
}) {
  const entries = useAllConditionals(character)
  const active = entries.some(
    (e) => e.active && e.effect.flag === spec.requiresFlag,
  )
  return (
    <span className={cn(CHIP, subtleText)}>
      Passiva · gatilho: {spec.requiresFlag}{' '}
      <span
        className={cn(
          active ? 'text-emerald-600 dark:text-emerald-400' : 'opacity-40',
        )}
        title={active ? `${spec.requiresFlag} ativa` : `${spec.requiresFlag} inativa`}
      >
        {active ? '●' : '○'}
      </span>
    </span>
  )
}

/**
 * Stance enter/exit lives HERE now (Phase 3) — the Efeitos tab only shows
 * the active card. Scaling stances open the stepper dialog; fixed-cost ones
 * activate in one tap (the success toast is the confirmation). While active,
 * a destructive-ghost "Encerrar" exits for free.
 */
function StanceSlot({
  spec,
  character,
}: {
  spec: ActivationSpec
  character: Character
}) {
  const entries = useAllConditionals(character)
  const { activateStance, deactivateStance } = usePowerAction(character)
  const flag = stanceFlagOf(spec)
  const group = entries.filter((e) => e.effect.flag === flag)
  const active = group.length > 0 && group.every((e) => e.active)
  const pmChip = (
    <span className={cn(CHIP, 'border-violet-500/40 text-violet-700 dark:text-violet-300')}>
      Postura · {spec.pmCost}{spec.scaling ? '+' : ''} PM
    </span>
  )
  if (active && flag) {
    return (
      <>
        {pmChip}
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="h-11 px-3 text-xs text-red-700 hover:bg-red-100 hover:text-red-700 dark:text-red-400 dark:hover:bg-red-950/40 sm:h-6 sm:px-2 sm:text-[11px]"
          onClick={() => deactivateStance(flag)}
          aria-label={`Encerrar ${spec.name}`}
        >
          ATIVA · Encerrar
        </Button>
      </>
    )
  }
  if (spec.scaling) {
    return (
      <>
        {pmChip}
        <UsePowerDialog spec={spec} character={character} />
      </>
    )
  }
  return (
    <>
      {pmChip}
      <FixedStanceActivateButton
        spec={spec}
        mpCurrent={character.mpCurrent}
        onActivate={() => activateStance(spec, 0)}
      />
    </>
  )
}

/** One-tap enter for stances without scaling — flat cost, no dialog. */
function FixedStanceActivateButton({
  spec,
  mpCurrent,
  onActivate,
}: {
  spec: ActivationSpec
  mpCurrent: number
  onActivate: () => void
}) {
  const decision = stanceActivationDecision(spec, 0, mpCurrent)
  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      className="h-11 px-3 text-xs sm:h-6 sm:px-2 sm:text-[11px]"
      disabled={!decision.ok}
      title={decision.reason}
      onClick={onActivate}
      aria-label={`Ativar ${spec.name}`}
    >
      <Zap className="mr-1 size-3" />
      Ativar {spec.pmCost} PM
    </Button>
  )
}

function InstantUseButton({
  spec,
  character,
}: {
  spec: ActivationSpec
  character: Character
}) {
  const { use, canUse } = usePowerAction(character)
  const { usedScene, usedDay } = usePowerUsedCounts(character.id, spec.id)
  const decision = canUse(spec)
  const scope = enforcedScopeOf(spec)
  const used = scope === 'scene' ? usedScene : usedDay
  return (
    <>
      {scope && (
        <span className={cn(CHIP, subtleText)}>
          usado {used}/1 {spec.uses === 'cena' ? 'cena' : 'dia'}
        </span>
      )}
      {!scope && usesBadge(spec.uses) && (
        <span className={cn(CHIP, subtleText)}>{usesBadge(spec.uses)}</span>
      )}
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="h-6 px-2 text-[11px]"
        disabled={!decision.ok}
        title={decision.reason}
        onClick={() => use(spec)}
        aria-label={`Usar ${spec.name}`}
      >
        <Zap className="mr-1 size-3" />
        Usar {spec.pmCost === 'variavel' ? '? PM' : `${spec.pmCost} PM`}
      </Button>
    </>
  )
}
