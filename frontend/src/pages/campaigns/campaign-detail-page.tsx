import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useSfx } from '@/shared/lib/use-sfx'
import { useSceneNav } from '@/shared/lib/use-scene-nav'
import { Skeleton } from '@/shared/ui/skeleton'
import {
  campaignQueryOptions,
  campaignMembersQueryOptions,
} from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import {
  CampaignTome,
  type CampaignTab,
} from '@/features/campaign-manage/campaign-tome'

const routeApi = getRouteApi('/campaigns/$id/')

/**
 * Campaign detail as a grimório scene (ALE-58) redesigned into an open tome
 * (ALE-59): the SceneShell owns the back-to-Crônicas control; the CampaignTome
 * renders the chronicle as a book (identity leaf + section leaf) with the
 * open-zoom/page-turn entrance. The active tab lives in `?tab=` so it deep-links
 * and survives the back button. The live-session route stays its own bare
 * match-mode scene.
 */
export function CampaignDetailPage() {
  const { id } = routeApi.useParams()
  const { tab: urlTab } = routeApi.useSearch()
  const navigate = useNavigate()
  const sfx = useSfx()
  const campaignId = Number(id)
  const campaign = useQuery(campaignQueryOptions(campaignId))
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const isGm = campaign.data?.role === 'gm'
  const activeSession = sessions.data?.find((s) => s.status === 'active')
  const playerCount =
    members.data?.filter((m) => m.role === 'player').length ?? 0

  // Local tab state drives the switch INSTANTLY (same fix as the ficha): the tab
  // used to live only in the URL, and TanStack's router state is a
  // useSyncExternalStore whose updates React can't defer, so every switch ran as
  // one synchronous navigation (the delay). Now a click / ↑↓ flips local state
  // (instant paint) and the URL reconciles in a passive, debounced effect AFTER
  // paint — off the switch's critical path. The URL stays the source for
  // deep-links + back (adopted below), so that behaviour is unchanged.
  const [tab, setTab] = useState<CampaignTab>(isTab(urlTab) ? urlTab : 'visao')
  useEffect(() => {
    const next: CampaignTab = isTab(urlTab) ? urlTab : 'visao'
    setTab((cur) => (cur === next ? cur : next))
  }, [urlTab])
  useEffect(() => {
    const current: CampaignTab = isTab(urlTab) ? urlTab : 'visao'
    if (current === tab) return
    const timer = setTimeout(() => {
      navigate({
        to: '.',
        search: (prev: Record<string, unknown>) => ({ ...prev, tab }),
        replace: true,
      })
    }, 250)
    return () => clearTimeout(timer)
  }, [tab, urlTab, navigate])
  const goToTab = (next: CampaignTab) => {
    if (next === tab) return
    sfx('select')
    setTab(next)
  }
  const back = () => {
    sfx('back')
    navigate({ to: '/campaigns' })
  }
  // Game-like walk through the chronicle via the shared scene-nav grammar: arrows
  // move the focus cursor by layout within a region and cross rail ↔ header ↔
  // content at the edges; PageUp/PageDown are the section bumpers; Esc pops
  // content → rail → back to the book. Sections switch through local `setTab`
  // only (never `navigate()` — that re-inflates the switch into a blocking task).
  const cycleSection = (delta: number) => {
    const tabs: CampaignTab[] = isGm
      ? ['visao', 'sessoes', 'membros', 'config']
      : ['visao', 'sessoes', 'membros']
    const i = Math.max(0, tabs.indexOf(tab))
    setTab(tabs[(i + delta + tabs.length) % tabs.length])
  }
  useSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-tome-root]'),
    onEscape: back,
    bumpers: { prev: () => cycleSection(-1), next: () => cycleSection(1) },
    sfx,
    active: !campaign.isLoading && !!campaign.data,
  })
  const resume = () => {
    if (!activeSession) return
    sfx('select')
    navigate({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: campaignId, sid: activeSession.id },
    })
  }

  if (campaign.isLoading)
    return (
      <SceneShell dense onBack={back}>
        <div className="w-full space-y-4">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-56 w-full" />
        </div>
      </SceneShell>
    )
  if (campaign.isError)
    return (
      <SceneShell dense onBack={back}>
        <p className="text-destructive">{(campaign.error as Error).message}</p>
      </SceneShell>
    )
  if (!campaign.data) return null

  return (
    <SceneShell dense onBack={back} onEnter={() => sfx('open')}>
      <CampaignTome
        campaign={campaign.data}
        campaignId={campaignId}
        isGm={isGm}
        activeSession={activeSession}
        playerCount={playerCount}
        current={tab}
        onTab={goToTab}
        onResume={resume}
      />
    </SceneShell>
  )
}

function isTab(v: string | undefined): v is CampaignTab {
  return v === 'visao' || v === 'sessoes' || v === 'membros' || v === 'config'
}
