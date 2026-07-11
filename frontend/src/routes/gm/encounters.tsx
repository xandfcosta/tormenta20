import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { EncounterBuilderPage } from '@/pages/gm/encounters-page'

export const Route = createFileRoute('/gm/encounters')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: EncounterBuilderPage,
})
