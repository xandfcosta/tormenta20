import { useQueryClient } from '@tanstack/solid-query'
import type { ActivationSpec } from '@/shared/api/catalog-types'
import { Flame, Minus, Plus, Zap } from 'lucide-solid'
import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { allConditionals } from '@/entities/character/derived'
import {
  stanceActivationDecision,
  stanceFlagOf,
  stanceMaxSteps,
  stanceTotalPm,
} from '@/entities/character/power-rules'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { cn } from '@/shared/lib/utils'
import { describeConditionalTarget } from './conditional-target-label'
import { ITEM_DIALOG_CONTENT, ItemDialogSection } from './item-dialog-kit'
import { usePowerActions } from './use-power-actions'
import { signed } from './signed'
import { FieldLabel } from '@/shared/ui/section-label'

/**
 * Activation dialog for SCALING stances (Fúria p40: base 2 PM, +1 PM per +1
 * bônus every 5 Bárbaro levels). Live total, pool check, primary disabled when
 * the pool can't pay. The stepped EXTRA is display-only — no engine modifier
 * backs it yet — so the preview labels it "(stepper)" and the paid steps land
 * in the stance-activation store.
 */
export function UsePowerDialog(props: {
  spec: ActivationSpec
  character: Character
  /** Custom trigger (HUD quick-bar chip). Defaults to the Poderes-tab button. */
  trigger?: (open: () => void) => JSX.Element
}) {
  const queryClient = useQueryClient()
  const actions = usePowerActions()
  const [open, setOpen] = createSignal(false)
  const [steps, setSteps] = createSignal(0)

  const maxSteps = () => stanceMaxSteps(props.spec, props.character.classes)
  const total = () => stanceTotalPm(props.spec, steps())
  const blocked = () =>
    !stanceActivationDecision(props.spec, steps(), props.character.mpCurrent).ok

  const activate = async () => {
    setOpen(false)
    await actions(queryClient, props.character).activateStance(props.spec, steps())
    setSteps(0)
  }

  const close = (next: boolean) => {
    setOpen(next)
    if (!next) setSteps(0)
  }

  return (
    <Show when={props.spec.scaling}>
      {(scaling) => (
        <>
          {props.trigger?.(() => setOpen(true)) ?? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              class="h-11 px-3 text-xs sm:h-6 sm:px-2 sm:text-2xs"
              aria-label={`Ativar ${props.spec.name}`}
              onClick={() => setOpen(true)}
            >
              <Zap aria-hidden="true" class="mr-1 size-3" />
              Ativar… {scaling().basePm}+ PM
            </Button>
          )}
          <Dialog open={open()} onOpenChange={close}>
            <DialogContent class={cn(ITEM_DIALOG_CONTENT, 'flex flex-col gap-3')}>
              <DialogHeader>
                <DialogTitle class="flex items-center gap-2 font-heading tracking-wide">
                  <Flame aria-hidden="true" class="size-5 text-[color:var(--primary)]" />
                  {props.spec.name}
                </DialogTitle>
                <DialogDescription>
                  Base {scaling().basePm} PM • PM atual {props.character.mpCurrent} /{' '}
                  {props.character.mpMax}
                </DialogDescription>
              </DialogHeader>

              <StepperRow
                stepLabel={scaling().stepLabel}
                stepPm={scaling().stepPm}
                steps={steps()}
                maxSteps={maxSteps()}
                onChange={setSteps}
              />
              <StancePreview spec={props.spec} character={props.character} steps={steps()} />
              <TotalRow
                total={total()}
                remaining={props.character.mpCurrent - total()}
                blocked={blocked()}
              />

              <DialogFooter>
                <Button variant="outline" class="min-h-11 sm:min-h-9" onClick={() => close(false)}>
                  Cancelar
                </Button>
                <Button class="min-h-11 sm:min-h-9" disabled={blocked()} onClick={() => void activate()}>
                  <Zap aria-hidden="true" class="mr-1 size-4" />
                  Ativar
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </>
      )}
    </Show>
  )
}

/** −/+ stepper (44px targets on phone) clamped to 0..maxStepsForLevel. */
function StepperRow(props: {
  stepLabel: string
  stepPm: number
  steps: number
  maxSteps: number
  onChange: (steps: number) => void
}) {
  return (
    <ItemDialogSection title="PM extra">
      <div class="flex items-center justify-between gap-2 rounded-none border border-border p-2">
        <div class="min-w-0 flex-1 space-y-0.5">
          <p class="text-xs text-foreground">{props.stepLabel}</p>
          <p class="text-2xs text-muted-foreground">
            +{props.stepPm} PM por passo · máx {props.maxSteps} pelo nível de classe
          </p>
        </div>
        <StepButton
          label="Diminuir passos"
          disabled={props.steps <= 0}
          onClick={() => props.onChange(Math.max(0, props.steps - 1))}
        >
          <Minus aria-hidden="true" class="size-4" />
        </StepButton>
        {/* role=status: a plain <span> has no role to carry `aria-label`, and
            the count must announce itself when +/- change it. */}
        <span
          role="status"
          aria-label={`Passos extras: ${props.steps}`}
          class="w-6 text-center font-mono text-lg font-bold text-foreground"
        >
          {props.steps}
        </span>
        <StepButton
          label="Aumentar passos"
          disabled={props.steps >= props.maxSteps}
          onClick={() => props.onChange(Math.min(props.maxSteps, props.steps + 1))}
        >
          <Plus aria-hidden="true" class="size-4" />
        </StepButton>
      </div>
    </ItemDialogSection>
  )
}

function StepButton(props: {
  label: string
  disabled: boolean
  onClick: () => void
  children: JSX.Element
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      class="size-11 shrink-0 sm:size-9"
      disabled={props.disabled}
      onClick={() => props.onClick()}
      aria-label={props.label}
    >
      {props.children}
    </Button>
  )
}

/**
 * What activating buys: the stance's flag-group modifiers (engine-backed), the
 * display-only stepper extra, and Alma de Bronze's temp PV when owned — the
 * sheet is computed with the stance FORCED on, so the dialog previews the
 * post-activation state rather than the current one.
 */
function StancePreview(props: { spec: ActivationSpec; character: Character; steps: number }) {
  const conditionals = useConditionals()
  const modifiers = createMemo(() => {
    const flag = stanceFlagOf(props.spec)
    return allConditionals(props.character, conditionals.active(props.character.id)).filter(
      (entry) => entry.effect.flag === flag,
    )
  })
  const tempHp = createMemo(() =>
    stanceFlagOf(props.spec) === 'furia'
      ? computedSheetFor(props.character, conditionals.active(props.character.id)).tempHpFuria
      : null,
  )

  return (
    <ItemDialogSection title="Efeito ao ativar">
      <ul class="space-y-0.5 text-2xs">
        <For each={modifiers()}>
          {(entry) => (
            <li class="flex items-center justify-between gap-2">
              <span class="truncate text-muted-foreground">
                {describeConditionalTarget(entry.effect.target)}
              </span>
              <span class="shrink-0 font-mono font-semibold text-bonus-ink">
                {signed(entry.effect.amount)}
              </span>
            </li>
          )}
        </For>
        <Show when={props.steps > 0}>
          <li class="italic text-muted-foreground">
            +{props.steps} extra (stepper) — anotação, não somado nos totais
          </li>
        </Show>
        <Show when={(tempHp()?.total ?? 0) > 0}>
          <li class="flex items-center justify-between gap-2">
            <span class="truncate text-muted-foreground">
              Alma de Bronze: PV temp = nível + For
            </span>
            <span class="shrink-0 font-mono font-semibold text-bonus-ink">
              +{tempHp()?.total}
            </span>
          </li>
        </Show>
      </ul>
    </ItemDialogSection>
  )
}

/** Live total + pool-after line; red when the pool can't pay. */
function TotalRow(props: { total: number; remaining: number; blocked: boolean }) {
  return (
    <div class="rounded-none border border-border bg-muted px-3 py-2">
      <div class="flex items-center justify-between">
        <FieldLabel class="text-xs">Custo total</FieldLabel>
        <span
          class={cn(
            'font-mono text-lg font-bold',
            props.blocked ? 'text-penalty-ink' : 'text-grimorio-gold',
          )}
        >
          {props.total} PM
        </span>
      </div>
      <p class={cn('text-2xs', props.blocked ? 'text-penalty-ink' : 'text-muted-foreground')}>
        {props.blocked
          ? 'PM insuficiente para ativar'
          : `PM restante após ativar: ${props.remaining}`}
      </p>
    </div>
  )
}
