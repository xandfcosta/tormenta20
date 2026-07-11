import { createFileRoute, redirect } from '@tanstack/react-router'
import { z } from 'zod'
import { charactersQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { JoinCampaignPage } from '@/pages/campaigns/campaign-join-page'

const joinSearchSchema = z.object({
  token: z.string().min(1).optional(),
})

export const Route = createFileRoute('/campaigns/join')({
  validateSearch: joinSearchSchema,
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) {
      throw redirect({ to: '/login', search: { redirect: location.href } })
    }
    await context.queryClient.ensureQueryData(charactersQueryOptions)
  },
  component: JoinCampaignPage,
})
