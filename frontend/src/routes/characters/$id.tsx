import { createFileRoute, redirect } from '@tanstack/react-router'
import { characterQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CharacterViewPage } from '@/pages/characters/character-editor-page'

export const Route = createFileRoute('/characters/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(characterQueryOptions(Number(params.id))),
  component: CharacterViewPage,
})
