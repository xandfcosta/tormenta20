import { createFileRoute, redirect } from '@tanstack/react-router'
import { characterOptionsQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { NewCharacterPage } from '@/pages/characters/character-wizard-page'

export const Route = createFileRoute('/characters/new')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(characterOptionsQueryOptions),
  component: NewCharacterPage,
})
