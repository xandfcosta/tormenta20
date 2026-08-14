import { useQuery } from '@tanstack/solid-query'
import { Link, getRouteApi, useNavigate } from '@tanstack/solid-router'
import { Show } from 'solid-js'
import { passwordResetQueryOptions } from '@/entities/user/queries'
import { api } from '@/shared/api/api'
import { AuthShell } from './auth-shell'
import { ResetPasswordForm } from './reset-password-form'

const route = getRouteApi('/redefinir-senha')

/**
 * A outra ponta do link que o admin gera (ALE-120): quem recebe escolhe a
 * PRÓPRIA senha.
 *
 * A tela pergunta pelo link ANTES de mostrar o formulário — um link vencido
 * dizer isso de cara é melhor que falhar no envio com a senha já digitada duas
 * vezes. E o e-mail que volta é a conferência de que é a conta certa.
 */
export function ResetPasswordPage() {
  const navigate = useNavigate()
  const search = route.useSearch()
  const target = useQuery(() => passwordResetQueryOptions(search().token))

  const submit = async (password: string) => {
    await api.auth.resetPassword({ token: search().token ?? '', password })
    await navigate({ to: '/login' })
  }

  return (
    <AuthShell
      title="Escolher nova senha"
      subtitle={target.data?.email ?? 'Um link de redefinição foi usado para chegar aqui.'}
      footer={
        <>
          Lembrou a senha?{' '}
          <Link to="/login" class="underline underline-offset-4">
            Entrar
          </Link>
        </>
      }
    >
      <Show when={target.isSuccess} fallback={<LinkProblem pending={target.isLoading} />}>
        <ResetPasswordForm onSubmit={submit} />
      </Show>
    </AuthShell>
  )
}

function LinkProblem(props: { pending: boolean }) {
  return (
    <p class="text-sm text-muted-foreground">
      <Show when={!props.pending} fallback="Conferindo o link…">
        Este link não vale mais — ele serve uma vez só e expira em 24 horas. Peça outro a quem
        administra a mesa.
      </Show>
    </p>
  )
}
