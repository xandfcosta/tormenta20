import { useQueryClient } from '@tanstack/solid-query'
import type { ActivationAction, ActivationSpec, ActivationUses } from '@/shared/api/catalog-types'
import { Zap } from 'lucide-solid'
import { Show, createMemo } from 'solid-js'
import { allConditionals } from '@/entities/character/derived'
import {
  enforcedScopeOf,
  stanceActivationDecision,
  stanceFlagOf,
} from '@/entities/character/power-rules'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { usePowerUses } from '@/shared/stores/power-uses-context'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import { UsePowerDialog } from './use-power-dialog'
import { usePowerActions } from './use-power-actions'

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

const CHIP =
  'shrink-0 rounded-full border border-border px-1.5 py-px text-4xs font-semibold uppercase tracking-wide text-muted-foreground'

function usesBadge(uses: ActivationUses): string | null {
  if (uses === null) return null
  if (typeof uses === 'number') return `${uses}/dia`
  return `1/${uses}`
}

/**
 * Inline use affordance for one ability row, rendered per the activation
 * taxonomy: passives get a chip, triggered passives a live gatilho dot,
 * instants the one-tap "Usar N PM", stances an Ativar (stepper dialog when the
 * stance scales). An `undefined` spec means the ability has no activation
 * entry — render nothing and leave the row as it is.
 */
export function PowerActionSlot(props: {
  spec: ActivationSpec | undefined
  character: Character
  class?: string
}) {
  return (
    <Show when={props.spec}>
      {(spec) => (
        <span class={cn('flex flex-wrap items-center gap-1', props.class)}>
          <Show when={spec().kind === 'passive'}>
            <span class={CHIP}>Passiva</span>
          </Show>
          <Show when={spec().kind === 'triggered-passive'}>
            <TriggeredPassiveChip spec={spec()} character={props.character} />
          </Show>
          <Show when={spec().kind === 'stance'}>
            <StanceSlot spec={spec()} character={props.character} />
          </Show>
          <Show when={spec().kind === 'instant'}>
            <ActionCostChip spec={spec()} />
            <InstantUseButton spec={spec()} character={props.character} />
          </Show>
        </span>
      )}
    </Show>
  )
}

/** `LIVRE · 1 PM`-style chip: what the use costs in action economy + PM. */
function ActionCostChip(props: { spec: ActivationSpec }) {
  const pm = () =>
    props.spec.pmCost === 'variavel' ? 'PM variável' : `${props.spec.pmCost} PM`
  return (
    <span class={CHIP}>
      {ACTION_LABEL[props.spec.action]} · {pm()}
    </span>
  )
}

/** Passive that only fires while its stance flag is up — live ●/○ dot. */
function TriggeredPassiveChip(props: { spec: ActivationSpec; character: Character }) {
  const conditionals = useConditionals()
  const active = createMemo(() =>
    allConditionals(props.character, conditionals.active(props.character.id)).some(
      (entry) => entry.active && entry.effect.flag === props.spec.requiresFlag,
    ),
  )
  return (
    <span class={CHIP}>
      Passiva · gatilho: {props.spec.requiresFlag}{' '}
      <span
        class={active() ? 'text-emerald-400' : 'opacity-40'}
        title={`${props.spec.requiresFlag} ${active() ? 'ativa' : 'inativa'}`}
      >
        {active() ? '●' : '○'}
      </span>
    </span>
  )
}

/**
 * Stance enter/exit lives HERE — the Efeitos block only shows the running
 * card. A scaling stance opens the stepper dialog; a fixed-cost one activates
 * in one tap (the success toast is the confirmation). While active, a
 * destructive-ghost "Encerrar" exits for free.
 */
function StanceSlot(props: { spec: ActivationSpec; character: Character }) {
  const queryClient = useQueryClient()
  const conditionals = useConditionals()
  const actions = usePowerActions()

  const flag = () => stanceFlagOf(props.spec)
  const active = createMemo(() => {
    const group = allConditionals(
      props.character,
      conditionals.active(props.character.id),
    ).filter((entry) => entry.effect.flag === flag())
    return group.length > 0 && group.every((entry) => entry.active)
  })

  return (
    <>
      <span
        class={cn(CHIP, 'border-violet-500/40 text-violet-300')}
      >
        Postura · {props.spec.pmCost}
        {props.spec.scaling ? '+' : ''} PM
      </span>
      <Show
        when={active() && flag()}
        fallback={
          <Show
            when={props.spec.scaling}
            fallback={
              <FixedStanceActivateButton
                spec={props.spec}
                mpCurrent={props.character.mpCurrent}
                onActivate={() =>
                  void actions(queryClient, props.character).activateStance(props.spec, 0)
                }
              />
            }
          >
            <UsePowerDialog spec={props.spec} character={props.character} />
          </Show>
        }
      >
        {(activeFlag) => (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            class="h-11 px-3 text-xs text-red-400 hover:bg-red-950/40 sm:h-6 sm:px-2 sm:text-2xs"
            aria-label={`Encerrar ${props.spec.name}`}
            onClick={() =>
              void actions(queryClient, props.character).deactivateStance(activeFlag())
            }
          >
            ATIVA · Encerrar
          </Button>
        )}
      </Show>
    </>
  )
}

/** One-tap enter for stances without scaling — flat cost, no dialog. */
function FixedStanceActivateButton(props: {
  spec: ActivationSpec
  mpCurrent: number
  onActivate: () => void
}) {
  const decision = () => stanceActivationDecision(props.spec, 0, props.mpCurrent)
  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      class="h-11 px-3 text-xs sm:h-6 sm:px-2 sm:text-2xs"
      disabled={!decision().ok}
      title={decision().reason}
      aria-label={`Ativar ${props.spec.name}`}
      onClick={() => props.onActivate()}
    >
      <Zap aria-hidden="true" class="mr-1 size-3" />
      Ativar {props.spec.pmCost} PM
    </Button>
  )
}

function InstantUseButton(props: { spec: ActivationSpec; character: Character }) {
  const queryClient = useQueryClient()
  const powerUses = usePowerUses()
  const actions = usePowerActions()

  const decision = () => actions(queryClient, props.character).canUse(props.spec)
  const scope = () => enforcedScopeOf(props.spec)
  const used = () => {
    const counts = powerUses.used(props.character.id, props.spec.id)
    return scope() === 'scene' ? counts.scene : counts.day
  }

  return (
    <>
      <Show when={scope()} fallback={<Show when={usesBadge(props.spec.uses)}>
        {(badge) => <span class={CHIP}>{badge()}</span>}
      </Show>}>
        <span class={CHIP}>
          usado {used()}/1 {props.spec.uses === 'cena' ? 'cena' : 'dia'}
        </span>
      </Show>
      <Button
        type="button"
        size="sm"
        variant="outline"
        class="h-11 px-3 text-xs sm:h-6 sm:px-2 sm:text-2xs"
        disabled={!decision().ok}
        title={decision().reason}
        aria-label={`Usar ${props.spec.name}`}
        onClick={() => void actions(queryClient, props.character).use(props.spec)}
      >
        <Zap aria-hidden="true" class="mr-1 size-3" />
        Usar {props.spec.pmCost === 'variavel' ? '? PM' : `${props.spec.pmCost} PM`}
      </Button>
    </>
  )
}
