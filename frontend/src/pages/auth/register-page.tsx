import { useQueryClient } from '@tanstack/solid-query'
import { Link, useNavigate } from '@tanstack/solid-router'
import { meQueryOptions } from '@/entities/user/queries'
import { api } from '@/shared/api/api'
import { AuthShell } from './auth-shell'
import { type RegisterInput, RegisterForm } from './register-form'

/** Route glue: creates the account, seeds the session cache, and moves on. */
export function RegisterPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const register = async (input: RegisterInput) => {
    const user = await api.auth.register(input)
    queryClient.setQueryData(meQueryOptions.queryKey, user)
    await navigate({ to: '/' })
  }

  return (
    <AuthShell
      title="Criar conta"
      subtitle="Junte-se à mesa."
      footer={
        <>
          Já tem conta?{' '}
          <Link to="/login" class="underline underline-offset-4">
            Entrar
          </Link>
        </>
      }
    >
      <RegisterForm onSubmit={register} />
    </AuthShell>
  )
}
