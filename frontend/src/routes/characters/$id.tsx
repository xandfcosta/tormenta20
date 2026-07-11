import { createFileRoute, Outlet, redirect } from '@tanstack/react-router'
import { characterQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'

/**
 * Layout route for `/characters/$id`. Renders only an <Outlet/> so the
 * editor index and the nested computed sheet (`/characters/$id/sheet`)
 * each own the full screen (the editor previously sat here and swallowed
 * the outlet). Character data is prefetched here so both children hydrate.
 */
export const Route = createFileRoute('/characters/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      characterQueryOptions(Number(params.id)),
    ),
  component: Outlet,
})
