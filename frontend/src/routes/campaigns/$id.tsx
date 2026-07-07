import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { PageChrome } from '@/components/ui/page-chrome'
import { Skeleton } from '@/components/ui/skeleton'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
  campaignSessionsQueryOptions,
  meQueryOptions,
} from '@/lib/queries'
import { CampaignHeaderCard } from './campaign-detail/header-card'
import { DeleteCampaignButton } from './campaign-detail/delete-campaign-button'
import { MembersCard } from './campaign-detail/members-card'
import { SessionsCard } from './campaign-detail/sessions-card'

export const Route = createFileRoute('/campaigns/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) => {
    const id = Number(params.id)
    return Promise.all([
      context.queryClient.ensureQueryData(campaignQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignSessionsQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignMembersQueryOptions(id)),
    ])
  },
  component: CampaignDetailPage,
})

function CampaignDetailPage() {
  const { id } = Route.useParams()
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
        <DeleteCampaignButton campaign={campaign.data} />
      </div>

      <CampaignHeaderCard campaign={campaign.data} />
      <MembersCard campaignId={campaignId} />
      <SessionsCard campaignId={campaignId} />
    </PageChrome>
  )
}
