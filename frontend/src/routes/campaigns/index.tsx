import { createFileRoute, redirect } from '@tanstack/react-router'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CampaignsListPage } from '@/pages/campaigns/campaign-list-page'

export const Route = createFileRoute('/campaigns/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(campaignsQueryOptions),
  component: CampaignsListPage,
})
