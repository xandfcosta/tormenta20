import { createFileRoute, redirect } from '@tanstack/react-router'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { SessionDetailPage } from '@/pages/sessions/session-tracker-page'
import { idParams } from '@/shared/lib/route-params'

export const Route = createFileRoute('/campaigns/$id/sessions/$sid')({
  params: idParams('sid'),
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      campaignSessionQueryOptions(params.id, params.sid),
    ),
  component: SessionDetailPage,
})
