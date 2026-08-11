import { useQuery } from '@tanstack/solid-query'
import { getRouteApi, useNavigate } from '@tanstack/solid-router'
import { Show } from 'solid-js'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import {
  type CampaignTab,
  CampaignTome,
  campaignTabs,
  isCampaignTab,
} from '@/features/campaign-manage/campaign-tome'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { Skeleton } from '@/shared/ui/skeleton'

const routeApi = getRouteApi('/campaigns/$id/')

/**
 * Campaign detail as a grimório scene, redesigned into an open tome: the
 * SceneShell owns the back-to-Crônicas control; the CampaignTome renders the
 * chronicle as a book. The active tab lives in `?tab=` so it deep-links and
 * survives the back button.
 *
 * The React version could NOT read the tab straight from the URL: its comment
 * explains that the router's state is a `useSyncExternalStore` React can't
 * defer, so every switch ran as one synchronous navigation and felt slow. It
 * kept a local `useState` mirror for the instant paint plus TWO effects — one
 * adopting the URL, one debouncing the URL write by 250ms — and the tab lived
 * in two places at once.
 *
 * Here the search param IS the state. No mirror, no debounce, no effects: with
 * no re-render to schedule, a navigation only updates what actually reads it.
 */
export function CampaignDetailPage() {
  const params = routeApi.useParams()
  const search = routeApi.useSearch()
  const navigate = useNavigate()
  const ui = useUi()
  const sfx = createSfx(ui)

  const campaignId = () => Number(params().id)
  const campaign = useQuery(() => campaignQueryOptions(campaignId()))
  const sessions = useQuery(() => campaignSessionsQueryOptions(campaignId()))
  const members = useQuery(() => campaignMembersQueryOptions(campaignId()))

  const isGm = () => campaign.data?.role === 'gm'
  const activeSession = () => sessions.data?.find((s) => s.status === 'active')
  const playerCount = () => members.data?.filter((m) => m.role === 'player').length ?? 0

  const tabs = () => campaignTabs(isGm())

  // `?tab=config` from a player's URL falls back instead of showing a section
  // their rail doesn't have (the real gate is the server's, this is UX).
  const tab = (): CampaignTab => {
    const fromUrl = search().tab
    return isCampaignTab(fromUrl) && tabs().includes(fromUrl) ? fromUrl : 'visao'
  }

  const goToTab = (next: CampaignTab) => {
    if (next === tab()) return
    sfx('select')
    navigate({ to: '.', search: { tab: next }, replace: true })
  }

  const back = () => {
    sfx('back')
    navigate({ to: '/campaigns' })
  }

  const resume = () => {
    const session = activeSession()
    if (!session) return
    sfx('select')
    navigate({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: String(campaignId()), sid: String(session.id) },
    })
  }

  // Game-like walk through the chronicle via the shared scene-nav grammar:
  // arrows move the focus cursor by layout within a region and cross rail ↔
  // header ↔ content at the edges; PageUp/PageDown are the section bumpers;
  // Esc pops content → rail → back to the book.
  const cycleSection = (delta: number) => {
    const sections = tabs()
    const index = Math.max(0, sections.indexOf(tab()))
    goToTab(sections[(index + delta + sections.length) % sections.length])
  }

  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-tome-root]'),
    onEscape: back,
    bumpers: { prev: () => cycleSection(-1), next: () => cycleSection(1) },
    sfx,
    active: () => !campaign.isLoading && !!campaign.data,
  })

  return (
    <Show
      when={!campaign.isLoading}
      fallback={
        <SceneShell dense onBack={back}>
          <div class="w-full space-y-4">
            <Skeleton class="h-8 w-56" />
            <Skeleton class="h-56 w-full" />
          </div>
        </SceneShell>
      }
    >
      <Show
        when={campaign.data}
        fallback={
          <SceneShell dense onBack={back}>
            <p class="text-destructive">{(campaign.error as Error | null)?.message}</p>
          </SceneShell>
        }
      >
        {(data) => (
          <SceneShell dense onBack={back} onEnter={() => sfx('open')}>
            <CampaignTome
              campaign={data()}
              campaignId={campaignId()}
              isGm={isGm()}
              activeSession={activeSession()}
              playerCount={playerCount()}
              current={tab()}
              onTab={goToTab}
              onResume={resume}
            />
          </SceneShell>
        )}
      </Show>
    </Show>
  )
}
