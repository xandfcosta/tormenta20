import { Outlet, useLocation, useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useEffect, useRef, useState } from 'react'
import { ATTRIBUTE_KEYS } from '@tormenta20/t20-data'
import { PageChrome } from '@/shared/ui/page-chrome'
import { Skeleton } from '@/shared/ui/skeleton'
import { SectionHeading } from '@/shared/ui/section-heading'
import { ApiError, api } from '@/shared/api/api'
import type { CreateCharacterInput } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { cn } from '@/shared/lib/utils'
import {
  characterOptionsQueryOptions,
  charactersQueryOptions,
} from '@/entities/character/queries'
import { CharacterPreviewRail } from '@/features/character-build/character-preview-rail'
import { CreationStepper } from '@/features/character-build/creation-stepper'
import { CreationWizardProvider } from '@/features/character-build/creation-wizard-context'
import { WizardFooterNav } from '@/features/character-build/wizard-footer-nav'
import { raceAttributeDeltas } from '@/features/character-build/grant-helpers'
import { totalSlots } from '@/features/character-build/class-power-helpers'
import {
  type CharacterFormValues,
  characterSchema,
  type StepSlug,
  WIZARD_STEPS,
  wizardDefaults,
} from '@/features/character-build/wizard-steps'
import { useCharacterDraftStore } from '@/features/character-build/character-draft-store'

/**
 * Layout shell for the multi-page creation wizard. Owns the single TanStack
 * Form instance + race-choice side-state + draft persistence, provides them to
 * every step via context, and frames the router Outlet with the stepper spine,
 * the live preview rail, and the sticky footer nav. Never unmounts across step
 * navigation, so form state persists.
 */
export function CreationWizardShell() {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const options = useQuery(characterOptionsQueryOptions)
  const [formError, setFormError] = useState<string | null>(null)

  // Draft state is a persisted Zustand store; the form hydrates from it once
  // and mirrors edits back so a refresh / leaving mid-flow survives.
  // Merge over defaults so a draft persisted before a field existed (e.g. an
  // old draft without classPowers/trainedExpertises) still hydrates every key.
  const initialValues = useRef({
    ...wizardDefaults,
    ...useCharacterDraftStore.getState().values,
  }).current
  const raceChoices = useCharacterDraftStore((s) => s.raceChoices)
  const setRaceChoices = useCharacterDraftStore((s) => s.setRaceChoices)
  const setValues = useCharacterDraftStore((s) => s.setValues)
  const resetDraft = useCharacterDraftStore((s) => s.reset)

  const form = useForm({
    defaultValues: initialValues,
    validators: { onSubmit: characterSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      // Fields hold the point-buy base; racial bonuses (incl. resolved
      // floating/subrace choices) are baked into the saved attributes here.
      const deltas = raceAttributeDeltas(
        value.races,
        useCharacterDraftStore.getState().raceChoices,
      )
      const attributes = Object.fromEntries(
        ATTRIBUTE_KEYS.map((k) => [k, value[k] + (deltas[k] ?? 0)]),
      ) as Record<(typeof ATTRIBUTE_KEYS)[number], number>
      const payload: CreateCharacterInput = {
        ...value,
        ...attributes,
        // Final guard: never save more elective powers than the class slots
        // earn (covers lowering the level then skipping the Poderes step).
        classPowers: (value.classPowers ?? []).slice(0, totalSlots(value.classes)),
        god: value.god ? value.god : undefined,
      }
      try {
        const created = await api.characters.create(payload)
        resetDraft()
        qc.invalidateQueries({ queryKey: charactersQueryOptions.queryKey })
        await navigate({ to: '/characters/$id', params: { id: created.id } })
      } catch (e) {
        if (!applyServerErrors(formApi, e) && e instanceof ApiError) {
          setFormError(e.message)
        } else if (!(e instanceof ApiError)) {
          setFormError('Unexpected error. Try again.')
        }
      }
    },
  })

  // Mirror form edits into the persisted draft store.
  useEffect(() => {
    const sub = form.store.subscribe(() =>
      setValues(form.state.values as CharacterFormValues),
    )
    return () => sub.unsubscribe()
  }, [form, setValues])

  const cancel = () => {
    resetDraft()
    navigate({ to: '/characters' })
  }

  const location = useLocation()
  const currentSlug = slugFromPath(location.pathname)

  if (options.isLoading) return <WizardSkeleton />
  if (!options.data)
    return (
      <PageChrome>
        <p className="text-destructive">Falha ao carregar opções</p>
      </PageChrome>
    )

  return (
    <CreationWizardProvider
      value={{
        form,
        options: options.data,
        raceChoices,
        setRaceChoices,
        formError,
        submit: () => form.handleSubmit(),
        cancel,
      }}
    >
      <PageChrome
        width="wide"
        padded={false}
        className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden px-4 py-4 sm:px-6"
      >
        <div className="shrink-0 space-y-4">
          <SectionHeading variant="kallyadranoch" as="h1">
            Novo personagem
          </SectionHeading>
          <CreationStepper current={currentSlug} />
        </div>
        <div
          className={cn(
            'grid min-h-0 flex-1 gap-6',
            // Resumo shows its own hero + full-width review, so drop the rail
            // (it would duplicate the hero splash).
            currentSlug !== 'resumo' && 'lg:grid-cols-[1fr_20rem]',
          )}
        >
          <div className="flex min-h-0 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto pr-1">
              <Outlet />
              {formError && (
                <p className="mt-3 text-sm text-destructive">{formError}</p>
              )}
            </div>
          </div>
          {currentSlug !== 'resumo' && (
            <aside className="hidden min-h-0 lg:block lg:self-start">
              <CharacterPreviewRail
                form={form}
                raceChoices={raceChoices}
                current={currentSlug}
              />
            </aside>
          )}
        </div>
        <WizardFooterNav current={currentSlug} />
      </PageChrome>
    </CreationWizardProvider>
  )
}

function slugFromPath(pathname: string): StepSlug {
  const last = pathname.split('/').filter(Boolean).at(-1)
  const match = WIZARD_STEPS.find((s) => s.slug === last)
  return match?.slug ?? 'raca'
}

function WizardSkeleton() {
  return (
    <PageChrome className="space-y-4">
      <Skeleton className="h-8 w-56" />
      <Skeleton className="h-10 w-full" />
      <div className="grid gap-4 sm:grid-cols-2">
        <Skeleton className="h-11 w-full" />
        <Skeleton className="h-11 w-full" />
      </div>
      <Skeleton className="h-40 w-full" />
    </PageChrome>
  )
}
