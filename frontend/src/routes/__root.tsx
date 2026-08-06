import {
  Outlet,
  createRootRouteWithContext,
  useNavigate,
  useRouterState,
} from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { AppShell } from '@/shared/layout/app-shell'
import { TooltipProvider } from '@/shared/ui/tooltip'
import { Toaster } from '@/shared/ui/sonner'
import { useUiStore } from '@/shared/stores/ui-store'
import { useAuthStore } from '@/shared/stores/auth-store'
import { meQueryOptions } from '@/entities/user/queries'
import { ensureCatalogs } from '@/entities/catalog/ensure-catalogs'
import { api } from '@/shared/api/api'

type RouterContext = { queryClient: QueryClient }

export const Route = createRootRouteWithContext<RouterContext>()({
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    // Prime the catalog cache once for the whole authenticated app (static,
    // cached-∞) so the sync accessors in derived.ts & co. are warm on ANY
    // authed route — not just the sheet/wizard. Skipped when logged out, so
    // login/register stay fast.
    if (user) await ensureCatalogs(context.queryClient)
    return { user }
  },
  component: RootLayout,
})

function RootLayout() {
  const { user } = Route.useRouteContext()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const setUser = useAuthStore((s) => s.setUser)
  const me = useQuery(meQueryOptions)
  const theme = useUiStore((s) => s.theme)
  const toggleTheme = useUiStore((s) => s.toggleTheme)
  // A live session runs full-screen ("match mode") — drop the app nav
  // when the session route is matched so the session bar owns the viewport.
  const inMatch = useRouterState({
    select: (s) =>
      s.matches.some(
        (m) => m.routeId === '/campaigns/$id/sessions/$sid',
      ),
  })
  // The character sheet stacks its own tab bar at the bottom edge; on phones
  // the app BottomNav directly beneath it made two same-height tap surfaces
  // ("Campanhas" existed in both). Suppress the app bar there — TopNav stays,
  // so navigation is never stranded (audit P1).
  const inSheet = useRouterState({
    select: (s) =>
      s.matches.some(
        (m) =>
          m.routeId === '/characters/$id/' ||
          m.routeId === '/characters/$id/sheet',
      ),
  })
  // Auth screens own the whole viewport too (split-screen AuthShell), so
  // they render in the bare shell with no app nav.
  const inAuth = useRouterState({
    select: (s) =>
      s.matches.some(
        (m) => m.routeId === '/login' || m.routeId === '/register',
      ),
  })

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark')
  }, [theme])

  useEffect(() => {
    setUser(me.data ?? user ?? null)
  }, [me.data, user, setUser])

  const logout = useMutation({
    mutationFn: api.auth.logout,
    onSuccess: async () => {
      qc.setQueryData(meQueryOptions.queryKey, null)
      qc.removeQueries({ queryKey: ['users'] })
      setUser(null)
      await navigate({ to: '/login' })
    },
  })

  const current = me.data ?? user ?? null

  return (
    <TooltipProvider delayDuration={150}>
      <AppShell
        user={current}
        theme={theme}
        onToggleTheme={toggleTheme}
        onLogout={() => logout.mutate()}
        logoutPending={logout.isPending}
        bare={inMatch || inAuth}
        hideBottomNav={inSheet}
      >
        <Outlet />
      </AppShell>
      <Toaster />
      {/* Bottom corners: the top ones sit over the TopNav on every route —
          the RQ toggle covered the brand ("Tormenta 20" read "menta 20") and
          the router pill hid the theme/user controls (UI audit task 14).
          Hidden below md: on phones the pills sat on top of the sheet's
          bottom tab bar, covering the last tabs. */}
      {import.meta.env.DEV && !inMatch && (
        <div className="hidden md:contents">
          <TanStackRouterDevtools position="bottom-right" />
          <ReactQueryDevtools buttonPosition="bottom-left" />
        </div>
      )}
    </TooltipProvider>
  )
}
