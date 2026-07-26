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
    <div className="-mx-4 flex shrink-0 items-center justify-between gap-2 border-t bg-card/95 px-4 py-3 sm:-mx-6 sm:px-6">
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
  )
}
