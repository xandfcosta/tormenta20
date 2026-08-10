import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useSfx } from '@/shared/lib/use-sfx'
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
  const { tab } = routeApi.useSearch()
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

  const current: CampaignTab = isTab(tab) ? tab : 'visao'
  const goToTab = (next: CampaignTab) =>
    navigate({ to: '.', search: { tab: next }, replace: true })
  const back = () => {
    sfx('select')
    navigate({ to: '/campaigns' })
  }
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
        <div className="mx-auto w-full max-w-5xl space-y-4">
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
    <SceneShell dense onBack={back} onEnter={() => sfx('transition')}>
      <CampaignTome
        campaign={campaign.data}
        campaignId={campaignId}
        isGm={isGm}
        activeSession={activeSession}
        playerCount={playerCount}
        current={current}
        onTab={goToTab}
        onResume={resume}
      />
    </SceneShell>
  )
}

function isTab(v: string | undefined): v is CampaignTab {
  return v === 'visao' || v === 'sessoes' || v === 'membros' || v === 'config'
}
