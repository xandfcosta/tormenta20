import { createFileRoute, redirect } from '@tanstack/react-router'
import { campaignMembersQueryOptions, campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CampaignDetailPage } from '@/pages/campaigns/campaign-detail-page'

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
