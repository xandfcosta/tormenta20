import { createFileRoute, redirect } from '@tanstack/react-router'
import { charactersQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CharactersListPage } from '@/pages/characters/character-list-page'

export const Route = createFileRoute('/characters/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) => context.queryClient.ensureQueryData(charactersQueryOptions),
  component: CharactersListPage,
})
