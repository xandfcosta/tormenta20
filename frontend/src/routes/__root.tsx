import type { QueryClient } from '@tanstack/solid-query'
import { Outlet, createRootRouteWithContext } from '@tanstack/solid-router'
import { createEffect } from 'solid-js'
import { ensureCatalogs, ensureEngineCatalogs } from '@/entities/catalog/ensure-catalogs'
import { meQueryOptions } from '@/entities/user/queries'
import { useAuth } from '@/shared/stores/auth-context'
import { useUi } from '@/shared/stores/ui-context'
import { Toaster } from '@/shared/ui/sonner'

export type RouterContext = { queryClient: QueryClient }

export const Route = createRootRouteWithContext<RouterContext>()({
  // Resolved once before any route renders, so guards read a settled session
  // instead of racing a loading state.
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    // The catalogs are fetched, not bundled, and the sheet derives through the
    // WASM engine — both have SYNCHRONOUS accessors, so they must be warm
    // before any sheet renders or `computedSheetFor` returns nothing and the
    // panel reads `.expertises` off undefined (ALE-83). Skipped when logged
    // out, so /login stays light; the two warm in parallel.
    if (user) {
      await Promise.all([
        ensureCatalogs(context.queryClient),
        ensureEngineCatalogs(context.queryClient),
      ])
    }
    return { user }
  },
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
