import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { Outlet, useNavigate } from '@tanstack/solid-router'
import { RotateCcw } from 'lucide-solid'
import { Show } from 'solid-js'
import { characterOptionsQueryOptions } from '@/entities/character/queries'
import { api } from '@/shared/api/api'
import { ForgeBeads } from '@/features/character-build/forge-beads'
import { ForgeFooter } from '@/features/character-build/forge-footer'
import { ForgeProvider } from '@/features/character-build/forge-context'
import { createForgeSubmit } from '@/features/character-build/forge-submit'
import { createCurrentStep } from '@/features/character-build/current-step'
import {
  type StepSlug,
  furthestReachableIndex,
  stepAt,
  wizardSteps,
} from '@/features/character-build/wizard-steps'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSfx } from '@/shared/lib/sfx'
import { createCharacterDraftStore } from '@/shared/stores/character-draft-store'
import { useUi } from '@/shared/stores/ui-context'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { Skeleton } from '@/shared/ui/skeleton'

/**
 * The Forja — the scene where a character is made. A full stage for the step
 * being answered, the progress necklace in the header, and the living preview
 * pinned to the bottom band, so a pick and the PV it buys are on screen at once.
 *
 * The shell owns the draft and never unmounts across steps: walking the wizard
 * changes the route, not the state. Leaving the scene KEEPS the draft (that is
 * what persisting it is for) — "Recomeçar" is the only way to throw it away.
 */
export function ForgePage() {
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)
  const options = useQuery(() => characterOptionsQueryOptions)
  // `create*`: the draft owns state between calls, so it is born ONCE here in
  // the component body — not per event (gotcha #17 of the port).
  const draft = createCharacterDraftStore()
  const queryClient = useQueryClient()
  // Same reason as the draft: it holds the in-flight guard, so it is born once
  // here and not per click (gotcha #17).
  const forge = createForgeSubmit({
    draft,
    queryClient,
    createCharacter: api.characters.create,
    onCreated: async (id) => {
      sfx('open')
      await navigate({ to: '/characters/$id', params: { id: String(id) } })
    },
  })

  const current = createCurrentStep()

  const goTo = (slug: StepSlug) => {
    sfx('select')
    navigate({ to: '/characters/new/$step', params: { step: slug } })
  }
  const step = (delta: -1 | 1) => {
    const target = stepAt(current(), delta, wizardSteps(draft.values, current()))
    if (target) goTo(target)
  }
  const restart = () => {
    draft.reset()
    goTo('raca')
  }

  return (
    <SceneShell
      dense
      bleed
      title="Forja"
      backLabel="Personagens"
      onBack={() => {
        sfx('back')
        navigate({ to: '/characters' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={
        <>
          <ForgeBeads
            steps={wizardSteps(draft.values, current())}
            current={current()}
            reachable={furthestReachableIndex(draft.values, draft.raceChoices)}
            onJump={goTo}
          />
          <ConfirmDialog
            title="Recomeçar a criação?"
            description="Todas as escolhas deste rascunho são descartadas e você volta ao primeiro passo. Não dá para desfazer."
            confirmLabel="Recomeçar"
            destructive
            onConfirm={restart}
            trigger={(open) => (
              // Rebaixado de propósito: ele APAGA o rascunho inteiro e estava
              // com o mesmo peso da trilha ao lado, que é o contrário do que
              // um botão destrutivo deve pesar (ALE-169). O diálogo de
              // confirmação continua sendo quem protege — isto é só parar de
              // convidar.
              <Button
                type="button"
                variant="ghost"
                size="sm"
                class="text-2xs text-muted-foreground/70 hover:text-foreground"
                onClick={open}
              >
                <RotateCcw aria-hidden="true" class="mr-1 size-3" />
                Recomeçar
              </Button>
            )}
          />
        </>
      }
    >
      <Show when={options.data} fallback={<ForgeSkeleton />}>
        {(loaded) => (
          <ForgeProvider draft={draft} options={loaded()}>
            <div class="flex min-h-0 flex-1 flex-col">
              <div class="flex min-h-0 flex-1 flex-col overflow-y-auto px-4 py-4">
                <Outlet />
              </div>
              <ForgeFooter
                draft={draft}
                current={current()}
                submitting={forge.isPending}
                error={forge.error}
                onStep={step}
                onCreate={() => forge.create()}
              />
            </div>
          </ForgeProvider>
        )}
      </Show>
    </SceneShell>
  )
}

function ForgeSkeleton() {
  return (
    <div class="flex flex-1 flex-col gap-3 p-4">
      <Skeleton class="h-6 w-48" />
      <div class="grid flex-1 gap-4 lg:grid-cols-[1.35fr_1fr]">
        <Skeleton class="h-full min-h-40 w-full" />
        <Skeleton class="h-full min-h-40 w-full" />
      </div>
    </div>
  )
}
