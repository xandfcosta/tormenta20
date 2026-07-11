import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { NewCampaignPage } from '@/pages/campaigns/campaign-new-page'

export const Route = createFileRoute('/campaigns/new')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: NewCampaignPage,
})
