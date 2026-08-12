import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { TextField } from '@/shared/ui/text-field'

export type RegisterInput = { email: string; password: string; name?: string }

const registerSchema = z
  .object({
    email: z.email('E-mail inválido'),
    name: z.string(),
    password: z.string().min(8, 'A senha precisa ter ao menos 8 caracteres'),
    confirm: z.string().min(1, 'Confirme sua senha'),
  })
  .refine((v) => v.password === v.confirm, {
    path: ['confirm'],
    message: 'As senhas não conferem',
  })

/**
 * Account creation form. Same contract as {@link LoginForm}: validation and
 * presentation here, the actual call injected — so it tests without a router.
 *
 * `confirm` never leaves this component; it only exists to catch a typo before
 * a password the player can't reproduce reaches the server.
 *
 * @example <RegisterForm onSubmit={(input) => api.auth.register(input)} />
 */
export function RegisterForm(props: { onSubmit: (input: RegisterInput) => Promise<void> }) {
  const [email, setEmail] = createSignal('')
  const [name, setName] = createSignal('')
  const [password, setPassword] = createSignal('')
  const [confirm, setConfirm] = createSignal('')
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [submitting, setSubmitting] = createSignal(false)

  const handleSubmit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = registerSchema.safeParse({
      email: email(),
      name: name(),
      password: password(),
      confirm: confirm(),
    })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setSubmitting(true)
    try {
      // An empty name is "no name", not the empty string.
      await props.onSubmit({
        email: parsed.data.email,
        password: parsed.data.password,
        name: parsed.data.name || undefined,
      })
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
        name="name"
        label="Nome (opcional)"
        autocomplete="name"
        value={name()}
        onInput={setName}
        errors={fieldErrors().name}
      />
      <TextField
        name="password"
        label="Senha"
        type="password"
        autocomplete="new-password"
        value={password()}
        onInput={setPassword}
        errors={fieldErrors().password}
        hint="Ao menos 8 caracteres."
      />
      <TextField
        name="confirm"
        label="Confirmar senha"
        type="password"
        autocomplete="new-password"
        value={confirm()}
        onInput={setConfirm}
        errors={fieldErrors().confirm}
      />

      <Show when={formError()}>
        {(message) => <p class="text-sm text-destructive">{message()}</p>}
      </Show>

      <Button type="submit" class="w-full" disabled={submitting()}>
        {submitting() ? 'Criando…' : 'Criar conta'}
      </Button>
    </form>
  )
}
