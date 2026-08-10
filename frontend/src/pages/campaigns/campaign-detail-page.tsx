import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { CalendarClock } from 'lucide-react'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useSfx } from '@/shared/lib/use-sfx'
import { Skeleton } from '@/shared/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { Card, CardContent } from '@/shared/ui/card'
import { campaignQueryOptions, campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { CampaignHeaderCard } from '@/features/campaign-manage/header-card'
import { CampaignOverview } from '@/features/campaign-manage/campaign-overview'
import { DeleteCampaignButton } from '@/features/campaign-manage/delete-campaign-button'
import { MembersCard } from '@/features/campaign-manage/members-card'
import { SessionsCard } from '@/features/campaign-manage/sessions-card'

const routeApi = getRouteApi('/campaigns/$id/')

type CampaignTab = 'visao' | 'sessoes' | 'membros' | 'config'

/**
 * Campaign detail (GM + player) as a grimório scene (ALE-58): a full-screen
 * SceneShell (back → Crônicas) over the name + status subtitle and the
 * Visão geral / Sessões / Membros / Config tabs. The active tab lives in
 * `?tab=` so it deep-links and survives the back button. Config (edit + delete)
 * is GM-only. The live-session route (`/sessions/$sid`) stays its own bare
 * match-mode scene; only this index page is the detail scene.
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

  if (campaign.isLoading)
    return (
      <SceneShell dense title="Campanha" onBack={back}>
        <div className="mx-auto w-full max-w-5xl space-y-4">
          <Skeleton className="h-8 w-56" />
          <Skeleton className="h-40 w-full" />
        </div>
      </SceneShell>
    )
  if (campaign.isError)
    return (
      <SceneShell dense title="Campanha" onBack={back}>
        <p className="text-destructive">{(campaign.error as Error).message}</p>
      </SceneShell>
    )
  if (!campaign.data) return null

  return (
    <SceneShell
      dense
      title={campaign.data.name}
      onBack={back}
      onEnter={() => sfx('transition')}
    >
      <div className="mx-auto w-full max-w-5xl space-y-5">
        <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
          <span>{isGm ? 'Mestrando' : 'Jogando'}</span>
          <span aria-hidden>·</span>
          <span className="flex items-center gap-1">
            <CalendarClock className="size-3" />
            Criada em{' '}
            {new Date(campaign.data.createdAt).toLocaleDateString('pt-BR')}
          </span>
          <span aria-hidden>·</span>
          <span>
            {playerCount} {playerCount === 1 ? 'jogador' : 'jogadores'}
          </span>
          {activeSession && (
            <>
              <span aria-hidden>·</span>
              <span className="flex items-center gap-1 font-medium text-foreground">
                <span className="size-2 rounded-full bg-[color:var(--hp-full)]" />
                Sessão {activeSession.sessionNumber} ativa
              </span>
            </>
          )}
        </p>

        <Tabs value={current} onValueChange={(v) => goToTab(v as CampaignTab)}>
          <TabsList className="max-w-full self-start overflow-x-auto [&>button]:shrink-0">
            <TabsTrigger value="visao">Visão geral</TabsTrigger>
            <TabsTrigger value="sessoes">Sessões</TabsTrigger>
            <TabsTrigger value="membros">Membros</TabsTrigger>
            {isGm && <TabsTrigger value="config">Config</TabsTrigger>}
          </TabsList>

          <TabsContent value="visao">
            <CampaignOverview
              campaignId={campaignId}
              isGm={isGm}
              onGoToTab={goToTab}
            />
          </TabsContent>
          <TabsContent value="sessoes">
            <SessionsCard campaignId={campaignId} isGm={isGm} />
          </TabsContent>
          <TabsContent value="membros">
            <MembersCard campaignId={campaignId} isGm={isGm} />
          </TabsContent>
          {isGm && (
            <TabsContent value="config" className="space-y-6">
              <CampaignHeaderCard campaign={campaign.data} />
              <Card className="border-destructive/40">
                <CardContent className="flex flex-wrap items-center justify-between gap-3 pt-6">
                  <div>
                    <p className="font-medium">Excluir campanha</p>
                    <p className="text-xs text-muted-foreground">
                      Remove todas as sessões e membros. Não pode ser desfeito.
                    </p>
                  </div>
                  <DeleteCampaignButton campaign={campaign.data} />
                </CardContent>
              </Card>
            </TabsContent>
          )}
        </Tabs>
      </div>
    </SceneShell>
  )
}

function isTab(v: string | undefined): v is CampaignTab {
  return v === 'visao' || v === 'sessoes' || v === 'membros' || v === 'config'
}
