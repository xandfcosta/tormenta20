import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { Link, getRouteApi, useNavigate } from '@tanstack/solid-router'
import { Show, createSignal } from 'solid-js'
import { joinTargetId } from '@/entities/campaign/join-target'
import { campaignsQueryOptions, inviteQueryOptions } from '@/entities/campaign/queries'
import { charactersQueryOptions } from '@/entities/character/queries'
import { HeroPicker } from '@/features/campaign-join/hero-picker'
import { InviteLetter } from '@/features/campaign-join/invite-letter'
import { api } from '@/shared/api/api'
import { SceneShell } from '@/shared/layout/scene-shell'
import { toSubmitFailure } from '@/shared/lib/form-errors'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { Button } from '@/shared/ui/button'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { TextField } from '@/shared/ui/text-field'
import { TomePage } from '@/shared/ui/tome-page'

const routeApi = getRouteApi('/campaigns/join')

/**
 * Entrar na mesa — the invite as a letter tucked into the grimório. Two ways in:
 * an invite link (`?token=…`, which names the campaign for you) or the campaign
 * number the GM read out loud.
 *
 * The React version had to MIRROR the resolved campaign id into form state
 * through an effect, with a `tokenApplied` flag to stop it clobbering manual
 * edits — doing it inline warned "Cannot update a component while rendering a
 * different component" (ALE-20). Here the id is simply derived from whichever
 * source is in play: no effect, no flag, nothing to get out of sync.
 */
export function CampaignJoinPage() {
  const search = routeApi.useSearch()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const ui = useUi()
  const sfx = createSfx(ui)

  const token = () => search().token
  const characters = useQuery(() => charactersQueryOptions)
  const invite = useQuery(() => inviteQueryOptions(token()))

  const [manualId, setManualId] = createSignal('')
  const [heroId, setHeroId] = createSignal<number | null>(null)
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const roster = () => characters.data ?? []
  const inviteLoading = () => !!token() && invite.isLoading
  const inviteInvalid = () => !!token() && invite.isError

  const campaignId = () =>
    joinTargetId({
      token: token(),
      invitedCampaignId: invite.data?.campaignId,
      typedId: manualId(),
    })

  const canJoin = () =>
    !pending() && !inviteLoading() && !inviteInvalid() && campaignId() !== null && heroId() !== null

  const back = () => {
    sfx('back')
    navigate({ to: '/campaigns' })
  }

  // Same grammar as every other scene: arrows walk the hero plates, Esc leaves.
  // The driver ignores keys while a field has focus, so typing stays native.
  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-tome-root]'),
    onEscape: back,
    sfx,
  })

  const join = async (event: SubmitEvent) => {
    event.preventDefault()
    const id = campaignId()
    const characterId = heroId()
    if (id === null || characterId === null) return
    setFormError(null)
    setPending(true)
    try {
      await api.members.add(id, {
        characterId,
        role: 'player',
        ...(token() ? { inviteToken: token() } : {}),
      })
      await queryClient.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
      await queryClient.invalidateQueries({ queryKey: ['characters', characterId, 'campaigns'] })
      sfx('open')
      await navigate({ to: '/campaigns/$id', params: { id: String(id) } })
    } catch (failure) {
      setFormError(toSubmitFailure(failure).formError ?? 'Não foi possível entrar na mesa.')
    } finally {
      setPending(false)
    }
  }

  return (
    <SceneShell dense onBack={back} onEnter={() => sfx('open')}>
      <TomePage>
        <JoinHeading />
        <form class="mx-auto w-full max-w-3xl space-y-6" onSubmit={join} noValidate>
          <Show when={token()}>
            <InviteLetter
              loading={inviteLoading()}
              invalid={inviteInvalid()}
              campaignName={invite.data?.campaignName}
            />
          </Show>

          <Show when={!token()}>
            <TextField
              name="campaignId"
              label="Número da campanha"
              type="number"
              value={manualId()}
              onInput={setManualId}
              hint="O mestre da mesa envia esse número."
            />
          </Show>

          <Show when={characters.isLoading}>
            <SkeletonRows count={2} />
          </Show>
          <Show when={!characters.isLoading && roster().length === 0}>
            <NoHeroes />
          </Show>
          <Show when={roster().length > 0}>
            <HeroPicker characters={roster()} selectedId={heroId()} onSelect={setHeroId} />
          </Show>

          <Show when={formError()}>
            {(message) => <p class="text-sm text-destructive">{message()}</p>}
          </Show>

          <div class="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={back}>
              Cancelar
            </Button>
            <Button type="submit" disabled={!canJoin()}>
              {pending() ? 'Entrando…' : 'Entrar na mesa'}
            </Button>
          </div>
        </form>
      </TomePage>
    </SceneShell>
  )
}

/** Illuminated head of the invite leaf. */
function JoinHeading() {
  return (
    <header class="mx-auto w-full max-w-3xl space-y-3 text-center">
      <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
        Carta de convite
      </p>
      <h1 class="font-display text-3xl uppercase leading-tight tracking-wide text-grimorio-gold sm:text-4xl">
        Entrar na mesa
      </h1>
      <div
        aria-hidden="true"
        class="h-px w-full bg-gradient-to-r from-transparent via-grimorio-gold/40 to-transparent"
      />
    </header>
  )
}

/** No characters yet — you can't sit at a table without someone to play. */
function NoHeroes() {
  return (
    <div class="flex flex-col items-center gap-3 rounded-sm border border-dashed border-grimorio-iron px-4 py-10 text-center">
      <p class="text-sm text-muted-foreground">
        Você ainda não tem heróis. Crie um antes de entrar numa mesa.
      </p>
      <Link to="/characters/new">
        <Button variant="outline" size="sm">
          Criar herói
        </Button>
      </Link>
    </div>
  )
}
