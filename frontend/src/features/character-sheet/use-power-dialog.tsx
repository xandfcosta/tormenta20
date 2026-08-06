import { useState } from 'react'
import { Flame, Minus, Plus, Zap } from 'lucide-react'
import type { ActivationSpec } from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import type { Character } from '@/shared/api/api'
import { useAllConditionals } from '@/entities/character/derived'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import {
  stanceActivationDecision,
  stanceFlagOf,
  stanceMaxSteps,
  stanceTotalPm,
  usePowerAction,
} from '@/entities/character/use-power-action'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { describeConditionalTarget } from './conditional-target-label'
import { ITEM_DIALOG_CONTENT, ItemDialogSection } from './item-dialog-kit'
import { signed } from './signed'

/**
 * Activation dialog for SCALING stances (Fúria p40: base 2 PM, +1 PM per +1
 * bônus every 5 Bárbaro levels). Structure cloned from CastSpellDialog: live
 * total, pool check, disabled primary when the pool can't pay. The stepped
 * EXTRA is display-only this phase (no engine modifier) — the preview labels
 * it "(stepper)" and the paid steps land in the stance-activation-store.
 */
export function UsePowerDialog({
  spec,
  character,
  trigger,
}: {
  spec: ActivationSpec
  character: Character
  /** Custom trigger (HUD quick-bar chip). Defaults to the Poderes-tab button. */
  trigger?: React.ReactNode
}) {
  const [open, setOpen] = useState(false)
  const [steps, setSteps] = useState(0)
  const { activateStance } = usePowerAction(character)
  const maxSteps = stanceMaxSteps(spec, character.classes)
  const total = stanceTotalPm(spec, steps)
  const blocked = !stanceActivationDecision(spec, steps, character.mpCurrent).ok

  const activate = () => {
    activateStance(spec, steps)
    setOpen(false)
    setSteps(0)
  }

  if (!spec.scaling) return null
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) setSteps(0)
      }}
    >
      <DialogTrigger asChild>
        {trigger ?? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="h-11 px-3 text-xs sm:h-6 sm:px-2 sm:text-[11px]"
            aria-label={`Ativar ${spec.name}`}
          >
            <Zap className="mr-1 size-3" />
            Ativar… {spec.scaling.basePm}+ PM
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className={cn(ITEM_DIALOG_CONTENT, 'flex flex-col gap-3')}>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 font-display tracking-wide">
            <Flame className="size-5 text-[color:var(--primary)]" />
            {spec.name}
          </DialogTitle>
          <DialogDescription>
            Base {spec.scaling.basePm} PM • PM atual {character.mpCurrent} /{' '}
            {character.mpMax}
          </DialogDescription>
        </DialogHeader>

        <StepperRow
          spec={spec}
          steps={steps}
          maxSteps={maxSteps}
          onChange={setSteps}
        />
        <StancePreview spec={spec} character={character} steps={steps} />
        <TotalRow
          total={total}
          remaining={character.mpCurrent - total}
          blocked={blocked}
        />

        <DialogFooter>
          <Button
            variant="outline"
            className="min-h-11 sm:min-h-9"
            onClick={() => setOpen(false)}
          >
            Cancelar
          </Button>
          <Button
            className="min-h-11 sm:min-h-9"
            disabled={blocked}
            onClick={activate}
          >
            <Zap className="mr-1 size-4" />
            Ativar
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

/** −/+ stepper (44px targets on phone) clamped to 0..maxStepsForLevel. */
function StepperRow({
  spec,
  steps,
  maxSteps,
  onChange,
}: {
  spec: ActivationSpec
  steps: number
  maxSteps: number
  onChange: (steps: number) => void
}) {
  const scaling = spec.scaling
  if (!scaling) return null
  return (
    <ItemDialogSection title="PM extra">
      <div className="flex items-center justify-between gap-2 rounded border border-border p-2">
        <div className="min-w-0 flex-1 space-y-0.5">
          <p className="text-xs text-foreground">{scaling.stepLabel}</p>
          <p className={cn('text-[11px]', dimText)}>
            +{scaling.stepPm} PM por passo · máx {maxSteps} pelo nível de classe
          </p>
        </div>
        <StepButton
          label="Diminuir passos"
          disabled={steps <= 0}
          onClick={() => onChange(Math.max(0, steps - 1))}
        >
          <Minus className="size-4" />
        </StepButton>
        {/* role=status: a plain span rejects aria-label (biome a11y) and the
            step count should announce itself when the +/- buttons change it. */}
        <span
          role="status"
          aria-label={`Passos extras: ${steps}`}
          className="w-6 text-center font-mono text-lg font-bold text-foreground"
        >
          {steps}
        </span>
        <StepButton
          label="Aumentar passos"
          disabled={steps >= maxSteps}
          onClick={() => onChange(Math.min(maxSteps, steps + 1))}
        >
          <Plus className="size-4" />
        </StepButton>
      </div>
    </ItemDialogSection>
  )
}

function StepButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string
  disabled: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      className="size-11 shrink-0 sm:size-9"
      disabled={disabled}
      onClick={onClick}
      aria-label={label}
    >
      {children}
    </Button>
  )
}

/**
 * What activating buys: the stance's flag-group modifiers (engine-backed),
 * the display-only stepper extra, and Alma de Bronze's temp PV when owned
 * (tempHpFromPowers with a hypothetical active=true — dialog previews the
 * post-activation state).
 */
function StancePreview({
  spec,
  character,
  steps,
}: {
  spec: ActivationSpec
  character: Character
  steps: number
}) {
  const entries = useAllConditionals(character)
  const sheet = useComputedSheet(character)
  const flag = stanceFlagOf(spec)
  const modifiers = entries.filter((e) => e.effect.flag === flag)
  const tempHp = flag === 'furia' ? sheet.tempHpFuria : null
  return (
    <ItemDialogSection title="Efeito ao ativar">
      <ul className="space-y-0.5 text-[11px]">
        {modifiers.map((e) => (
          <li key={e.id} className="flex items-center justify-between gap-2">
            <span className={cn('truncate', dimText)}>
              {describeConditionalTarget(e.effect.target)}
            </span>
            <span className="shrink-0 font-mono font-semibold text-emerald-700 dark:text-emerald-300">
              {signed(e.effect.amount)}
            </span>
          </li>
        ))}
        {steps > 0 && (
          <li className={cn('italic', dimText)}>
            +{steps} extra (stepper) — anotação, não somado nos totais
          </li>
        )}
        {tempHp && tempHp.total > 0 && (
          <li className="flex items-center justify-between gap-2">
            <span className={cn('truncate', dimText)}>
              Alma de Bronze: PV temp = nível + For
            </span>
            <span className="shrink-0 font-mono font-semibold text-emerald-700 dark:text-emerald-300">
              +{tempHp.total}
            </span>
          </li>
        )}
      </ul>
    </ItemDialogSection>
  )
}

/** Live total + pool-after line; red when the pool can't pay (cast-dialog idiom). */
function TotalRow({
  total,
  remaining,
  blocked,
}: {
  total: number
  remaining: number
  blocked: boolean
}) {
  return (
    <div className="rounded-lg border border-border bg-muted px-3 py-2">
      <div className="flex items-center justify-between">
        <span className={cn('text-xs uppercase tracking-widest', dimText)}>
          Custo total
        </span>
        <span
          className={cn(
            'font-mono text-lg font-bold',
            blocked ? 'text-red-700 dark:text-red-400' : accentStrong,
          )}
        >
          {total} PM
        </span>
      </div>
      <p
        className={cn(
          'text-[11px]',
          blocked ? 'text-red-700 dark:text-red-400' : dimText,
        )}
      >
        {blocked
          ? 'PM insuficiente para ativar'
          : `PM restante após ativar: ${remaining}`}
      </p>
    </div>
  )
}
