import { useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import type { CampaignFormValues } from '@/entities/campaign/campaign-schema'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { CampaignForm } from '@/features/campaign-manage/campaign-form'
import { api } from '@/shared/api/api'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { TomeHeading } from '@/shared/ui/tome-heading'
import { TomePage } from '@/shared/ui/tome-page'

/**
 * Nova campanha as a BLANK LEAF of the grimório: the same leather and page-turn
 * as the chronicle you read, but the page is empty and you're the one writing
 * it. Creating turns straight to the new chronicle's page.
 */
export function NewCampaignPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const ui = useUi()
  const sfx = createSfx(ui)

  const back = () => {
    sfx('back')
    navigate({ to: '/campaigns' })
  }

  // Esc leaves the leaf, like every other scene. The driver stands down while a
  // field has focus, so it never eats what the player is typing.
  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-tome-root]'),
    onEscape: back,
    sfx,
  })

  const create = async (values: CampaignFormValues) => {
    // The backend takes description as optional; an empty box means "none",
    // not "an empty description".
    const created = await api.campaigns.create({
      name: values.name,
      ...(values.description ? { description: values.description } : {}),
    })
    await queryClient.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
    sfx('open')
    await navigate({ to: '/campaigns/$id', params: { id: String(created.id) } })
  }

  return (
    <SceneShell dense onBack={back} onEnter={() => sfx('open')}>
      <TomePage>
        {/* `m-auto`: a blank leaf holds little, so the entry sits in the MIDDLE
            of the page instead of clinging to the top of a vast empty rectangle
            on a desktop screen. With taller content the auto margins collapse. */}
        <div class="m-auto w-full max-w-2xl space-y-6">
          <TomeHeading eyebrow="Folha em branco" icon={<BlankLeafMark />}>
            Abrir nova crônica
          </TomeHeading>
          <CampaignForm
            submitLabel="Abrir crônica"
            pendingLabel="Abrindo…"
            onSubmit={create}
            onCancel={back}
          />
        </div>
      </TomePage>
    </SceneShell>
  )
}

/** The scribe's mark on an unwritten page. */
function BlankLeafMark() {
  return (
    <span aria-hidden="true" class="mr-2 text-grimorio-gold/60">
      ✦
    </span>
  )
}
