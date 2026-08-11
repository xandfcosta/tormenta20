import { useQueryClient } from '@tanstack/solid-query'
import { getRouteApi, useNavigate } from '@tanstack/solid-router'
import { meQueryOptions } from '@/entities/user/queries'
import { type Credentials, api } from '@/shared/api/api'
import { AuthShell } from './auth-shell'
import { LoginForm } from './login-form'

const routeApi = getRouteApi('/login')

/** Route glue: logs in, seeds the session cache, and moves on. */
export function LoginPage() {
  const navigate = useNavigate()
  const search = routeApi.useSearch()
  const queryClient = useQueryClient()

  const login = async (credentials: Credentials) => {
    const user = await api.auth.login(credentials)
    // Seed instead of invalidate: the guard on the next route reads this
    // immediately, and a refetch would race the navigation.
    queryClient.setQueryData(meQueryOptions.queryKey, user)
    await navigate({ to: search().redirect ?? '/' })
  }

  return (
    // The "Criar uma conta" link returns with /register, ported in ALE-69.
    <AuthShell title="Entrar" subtitle="Bem-vindo de volta, aventureiro.">
      <LoginForm onSubmit={login} />
    </AuthShell>
  )
}
