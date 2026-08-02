import { createFileRoute, redirect } from '@tanstack/react-router'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { CharacterSheetPage } from '@/pages/characters/character-sheet-page'

export const Route = createFileRoute('/characters/$id/sheet')({
  // Sheet tab lives in the URL so tab-eviction/mis-nav doesn't lose context.
  validateSearch: (search: Record<string, unknown>): { tab?: string } =>
    typeof search.tab === 'string' ? { tab: search.tab } : {},
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(characterSheetQueryOptions(params.id)),
  component: CharacterSheetPage,
})
