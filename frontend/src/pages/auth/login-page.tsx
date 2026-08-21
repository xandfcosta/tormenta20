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
    <AuthShell
      title="Entrar"
      subtitle="Bem-vindo de volta, aventureiro."
      // Não há link para criar conta, e a ausência é a decisão (ALE-120): esta
      // mesa é por convite, e quem tem um chega pela URL do convite, não por
      // aqui. O link que existia era resto do template — ele anunciava uma
      // porta que o servidor recusa com 403 para todo mundo menos o dono.
      footer={<>Esta mesa é por convite — peça o link a quem administra.</>}
    >
      <LoginForm onSubmit={login} />
    </AuthShell>
  )
}
