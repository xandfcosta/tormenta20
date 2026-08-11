import type { QueryClient } from '@tanstack/solid-query'
import { Outlet, createRootRouteWithContext } from '@tanstack/solid-router'
import { createEffect } from 'solid-js'
import { meQueryOptions } from '@/entities/user/queries'
import { useAuth } from '@/shared/stores/auth-context'
import { useUi } from '@/shared/stores/ui-context'
import { Toaster } from '@/shared/ui/sonner'

export type RouterContext = { queryClient: QueryClient }

export const Route = createRootRouteWithContext<RouterContext>()({
  // Resolved once before any route renders, so guards read a settled session
  // instead of racing a loading state.
  beforeLoad: async ({ context }) => ({
    user: await context.queryClient.ensureQueryData(meQueryOptions),
  }),
  component: RootLayout,
})

/**
 * Thin root: mirrors the resolved session into the auth store and renders the
 * matched route. The app shell (nav, theme, toaster) lands in ALE-66.
 */
function RootLayout() {
  const context = Route.useRouteContext()
  const auth = useAuth()
  const ui = useUi()
  createEffect(() => auth.setUser(context().user))
  return (
    <>
      <Outlet />
      <Toaster theme={ui.theme()} />
    </>
  )
}
