import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { RandomTablesPage } from '@/pages/gm/random-tables-page'

export const Route = createFileRoute('/gm/random-tables')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: RandomTablesPage,
})
