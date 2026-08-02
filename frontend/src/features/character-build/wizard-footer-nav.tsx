import { useNavigate } from '@tanstack/react-router'
import { Button } from '@/shared/ui/button'
import { useCreationWizard } from './creation-wizard-context'
import {
  allStepsReady,
  type CharacterFormValues,
  stepIndex,
  stepReady,
  type StepSlug,
  WIZARD_STEPS,
} from './wizard-steps'

/**
 * Sticky footer navigation. Voltar/Cancelar on the left, Próximo/Criar on the
 * right. "Próximo" is gated by the current step's readiness; on the Resumo the
 * primary action becomes "Criar personagem", gated by every step being ready.
 */
export function WizardFooterNav({ current }: { current: StepSlug }) {
  const { form, raceChoices, submit, cancel } = useCreationWizard()
  const navigate = useNavigate()
  const idx = stepIndex(current)
  const prev = WIZARD_STEPS[idx - 1]
  const next = WIZARD_STEPS[idx + 1]
  const isLast = current === 'resumo'
  const go = (slug: StepSlug) =>
    navigate({ to: `/characters/new/${slug}` as string })

  return (
    // Full-bleed fixed bar: the wizard PageChrome is a centered max-w
    // container, so an in-flow bar leaves page-background gaps on wide
    // viewports (sides beyond the max-w) and below (container py). Fixed to
    // the viewport with the content re-centered inside.
    <div className="fixed inset-x-0 bottom-0 z-10 border-t bg-card/95 backdrop-blur">
      <div className="mx-auto flex w-full max-w-[90rem] items-center justify-between gap-2 px-4 py-3 sm:px-6">
      {prev ? (
        <Button type="button" variant="outline" onClick={() => go(prev.slug)}>
          ‹ Voltar
        </Button>
      ) : (
        <Button type="button" variant="outline" onClick={cancel}>
          Cancelar
        </Button>
      )}

      <form.Subscribe
        selector={(s: {
          values: CharacterFormValues
          isSubmitting: boolean
        }) => ({ values: s.values, isSubmitting: s.isSubmitting })}
      >
        {({
          values,
          isSubmitting,
        }: {
          values: CharacterFormValues
          isSubmitting: boolean
        }) =>
          isLast ? (
            <Button
              type="button"
              onClick={submit}
              disabled={isSubmitting || !allStepsReady(values, raceChoices)}
            >
              {isSubmitting ? 'Criando…' : 'Criar personagem'}
            </Button>
          ) : (
            <Button
              type="button"
              onClick={() => next && go(next.slug)}
              disabled={!stepReady(current, values, raceChoices)}
            >
              Próximo ›
            </Button>
          )
        }
      </form.Subscribe>
      </div>
    </div>
  )
}
