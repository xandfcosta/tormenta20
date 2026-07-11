import { createFileRoute, redirect } from '@tanstack/react-router'
import { meQueryOptions } from '@/entities/user/queries'
import { DungeonGeneratorPage } from '@/pages/gm/dungeon-generator-page'

export const Route = createFileRoute('/gm/dungeon-generator')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  component: DungeonGeneratorPage,
})
