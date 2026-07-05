import { Outlet, createRootRouteWithContext, Link, useNavigate } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { useMutation, useQuery, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { TooltipProvider } from '@/components/ui/tooltip'
import { useUiStore } from '@/store/ui-store'
import { useAuthStore } from '@/store/auth-store'
import { meQueryOptions } from '@/lib/queries'
import { api } from '@/lib/api'

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
      <div className="flex h-dvh flex-col overflow-x-hidden bg-background text-foreground">
      <header className="flex items-center justify-between border-b px-6 py-3">
        <nav className="flex gap-4">
          <Link to="/" className="font-medium hover:underline">Home</Link>
          {current && (
            <>
              <Link to="/characters" className="font-medium hover:underline">Characters</Link>
              <Link to="/campaigns" className="font-medium hover:underline">Campanhas</Link>
              <Link to="/users" className="font-medium hover:underline">Users</Link>
            </>
          )}
        </nav>
        <div className="flex items-center gap-3">
          <Button variant="outline" size="sm" onClick={toggleTheme}>
            {theme === 'light' ? 'Dark' : 'Light'} mode
          </Button>
          {current ? (
            <>
              <span className="text-sm text-muted-foreground">{current.name ?? current.email}</span>
              <Button size="sm" onClick={() => logout.mutate()} disabled={logout.isPending}>
                Logout
              </Button>
            </>
          ) : (
            <>
              <Link to="/login">
                <Button variant="outline" size="sm">Login</Button>
              </Link>
              <Link to="/register">
                <Button size="sm">Register</Button>
              </Link>
            </>
          )}
        </div>
      </header>
      <main className="min-h-0 flex-1 overflow-hidden">
        <Outlet />
      </main>
      <TanStackRouterDevtools position="bottom-right" />
      <ReactQueryDevtools buttonPosition="bottom-left" />
      </div>
    </TooltipProvider>
  )
}
