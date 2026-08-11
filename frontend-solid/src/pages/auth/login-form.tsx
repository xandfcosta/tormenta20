import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import type { Credentials } from '@/shared/api/api'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { TextField } from '@/shared/ui/text-field'

const loginSchema = z.object({
  email: z.email('E-mail inválido'),
  password: z.string().min(1, 'Informe sua senha'),
})

/**
 * Login form — pure presentation + validation. Auth/routing is the caller's
 * job, injected as `onSubmit`, so this renders in a test with no router or
 * query client (the React version needed both).
 *
 * @example <LoginForm onSubmit={(c) => api.auth.login(c)} />
 */
export function LoginForm(props: { onSubmit: (credentials: Credentials) => Promise<void> }) {
  const [email, setEmail] = createSignal('')
  const [password, setPassword] = createSignal('')
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [submitting, setSubmitting] = createSignal(false)

  const handleSubmit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = loginSchema.safeParse({ email: email(), password: password() })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setSubmitting(true)
    try {
      await props.onSubmit(parsed.data)
    } catch (error) {
      const failure = toSubmitFailure(error)
      setFieldErrors(failure.fieldErrors)
      setFormError(failure.formError)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form class="space-y-4" onSubmit={handleSubmit} noValidate>
      <TextField
        name="email"
        label="E-mail"
        type="email"
        autocomplete="email"
        value={email()}
        onInput={setEmail}
        errors={fieldErrors().email}
      />
      <TextField
        name="password"
        label="Senha"
        type="password"
        autocomplete="current-password"
        value={password()}
        onInput={setPassword}
        errors={fieldErrors().password}
      />

      <Show when={formError()}>
        {(message) => <p class="text-sm text-destructive">{message()}</p>}
      </Show>

      <Button type="submit" class="w-full" disabled={submitting()}>
        {submitting() ? 'Entrando…' : 'Entrar'}
      </Button>
    </form>
  )
}
