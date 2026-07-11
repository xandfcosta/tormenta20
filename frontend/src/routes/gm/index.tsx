import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { GMHubPage } from '@/pages/gm/gm-hub-page'

export const Route = createFileRoute('/gm/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: GMHubPage,
})
