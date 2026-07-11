import { Outlet, createRootRouteWithContext, useNavigate } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { AppShell } from '@/shared/layout/app-shell'
import { TooltipProvider } from '@/shared/ui/tooltip'
import { useUiStore } from '@/shared/stores/ui-store'
import { useAuthStore } from '@/shared/stores/auth-store'
import { meQueryOptions } from '@/entities/user/queries'
import { api } from '@/shared/api/api'

type RouterContext = { queryClient: QueryClient }

export const Route = createRootRouteWithContext<RouterContext>()({
  beforeLoad: async ({ context }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
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
      >
        <Outlet />
      </AppShell>
      <TanStackRouterDevtools position="bottom-right" />
      <ReactQueryDevtools buttonPosition="bottom-left" />
    </TooltipProvider>
  )
}
