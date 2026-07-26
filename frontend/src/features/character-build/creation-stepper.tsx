import { useNavigate } from '@tanstack/react-router'
import { Check } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { useCreationWizard } from './creation-wizard-context'
import {
  type CharacterFormValues,
  furthestReachableIndex,
  stepIndex,
  type StepSlug,
  WIZARD_STEPS,
} from './wizard-steps'

/**
 * Progress spine for the creation wizard. A numbered node per step: done
 * (check, clickable jump-back), current (hue-tinted), or locked (until its
 * predecessors validate). Collapses to "Passo N de M · Label" + a progress bar
 * on mobile. Reachability is derived from the live form — no duplicated state.
 */
export function CreationStepper({ current }: { current: StepSlug }) {
  const { form, raceChoices } = useCreationWizard()
  return (
    <form.Subscribe selector={(s: { values: CharacterFormValues }) => s.values}>
      {(values: CharacterFormValues) => (
        <StepperBody
          current={current}
          reachable={furthestReachableIndex(values, raceChoices)}
        />
      )}
    </form.Subscribe>
  )
}

function StepperBody({
  current,
  reachable,
}: {
  current: StepSlug
  reachable: number
}) {
  const navigate = useNavigate()
  const currentIdx = stepIndex(current)
  const go = (slug: StepSlug) =>
    navigate({ to: `/characters/new/${slug}` as string })

  return (
    <>
      {/* Desktop: full numbered spine */}
      <ol className="hidden items-center gap-1 sm:flex">
        {WIZARD_STEPS.map((step, i) => {
          const state =
            i === currentIdx ? 'current' : i < currentIdx ? 'done' : 'locked'
          const clickable = i <= reachable && i !== currentIdx
          return (
            <li key={step.slug} className="flex items-center gap-1">
              <button
                type="button"
                disabled={!clickable}
                aria-current={state === 'current' ? 'step' : undefined}
                onClick={() => clickable && go(step.slug)}
                className={cn(
                  'flex items-center gap-2 rounded-md px-2 py-1 text-sm transition-colors',
                  state === 'current' && 'font-semibold text-foreground',
                  state !== 'current' && 'text-muted-foreground',
                  clickable && 'hover:bg-accent hover:text-foreground',
                  !clickable && i > reachable && 'opacity-50',
                )}
              >
                <span
                  className={cn(
                    'flex size-6 shrink-0 items-center justify-center rounded-full border text-xs',
                    state === 'current' && 'border-primary text-primary',
                    state === 'done' &&
                      'border-primary bg-primary text-primary-foreground',
                    state === 'locked' && 'border-border',
                  )}
                >
                  {state === 'done' ? <Check className="size-3.5" /> : i + 1}
                </span>
                {step.label}
              </button>
              {i < WIZARD_STEPS.length - 1 && (
                <span className="h-px w-3 bg-border" aria-hidden />
              )}
            </li>
          )
        })}
      </ol>

      {/* Mobile: compact position + progress bar */}
      <div className="sm:hidden">
        <p className="mb-1 text-sm font-medium">
          Passo {currentIdx + 1} de {WIZARD_STEPS.length} ·{' '}
          {WIZARD_STEPS[currentIdx].label}
        </p>
        <div className="h-1 overflow-hidden rounded-full bg-border">
          <div
            className="h-full bg-primary transition-all"
            style={{
              width: `${((currentIdx + 1) / WIZARD_STEPS.length) * 100}%`,
            }}
          />
        </div>
      </div>
    </>
  )
}
