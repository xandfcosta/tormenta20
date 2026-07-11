import { getRouteApi } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/ui/button'
import { PageChrome } from '@/shared/ui/page-chrome'
import { Skeleton } from '@/shared/ui/skeleton'
import { campaignQueryOptions } from '@/entities/campaign/queries'
import { CampaignHeaderCard } from '@/features/campaign-manage/header-card'
import { DeleteCampaignButton } from '@/features/campaign-manage/delete-campaign-button'
import { InviteButton } from '@/features/campaign-manage/invite-button'
import { MembersCard } from '@/features/campaign-manage/members-card'
import { SessionsCard } from '@/features/campaign-manage/sessions-card'

const routeApi = getRouteApi('/campaigns/$id')

export function CampaignDetailPage() {
  const { id } = routeApi.useParams()
  const campaignId = Number(id)
  const campaign = useQuery(campaignQueryOptions(campaignId))

  if (campaign.isLoading)
    return (
      <PageChrome width="wide" className="space-y-4">
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-32 w-full" />
      </PageChrome>
    )
  if (campaign.isError)
    return (
      <PageChrome width="wide">
        <p className="text-destructive">
          {(campaign.error as Error).message}
        </p>
      </PageChrome>
    )
  if (!campaign.data) return null

  return (
    <PageChrome width="wide" className="space-y-6">
      <div className="flex items-center justify-between">
        <Link to="/campaigns">
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
        <div className="flex items-center gap-2">
          <InviteButton campaignId={campaignId} />
          <DeleteCampaignButton campaign={campaign.data} />
        </div>
      </div>

      <CampaignHeaderCard campaign={campaign.data} />
      <MembersCard campaignId={campaignId} />
      <SessionsCard campaignId={campaignId} />
    </PageChrome>
  )
}
