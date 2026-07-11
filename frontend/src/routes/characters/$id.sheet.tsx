import { createFileRoute, redirect } from '@tanstack/react-router'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CharacterSheetPage } from '@/pages/characters/character-sheet-page'

export const Route = createFileRoute('/characters/$id/sheet')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      characterSheetQueryOptions(Number(params.id)),
    ),
  component: CharacterSheetPage,
})
