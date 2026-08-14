import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { ApiError } from '@/shared/api/api'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { TextField } from '@/shared/ui/text-field'

export type RegisterInput = {
  email: string
  password: string
  name?: string
  /** The single-use link from `?convite=`, when the player arrived with one. */
  inviteToken?: string
}

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
 * The form submits with or without an invite: the ADMIN_EMAILS address creates
 * the first account on a fresh machine and has no link to carry, so who may
 * register is the SERVER's answer, never a guess made here (ALE-120).
 *
 * @example <RegisterForm onSubmit={(input) => api.auth.register(input)} />
 */
export function RegisterForm(props: {
  onSubmit: (input: RegisterInput) => Promise<void>
  inviteToken?: string
}) {
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
        inviteToken: props.inviteToken,
      })
    } catch (error) {
      const failure = toSubmitFailure(error)
      setFieldErrors(failure.fieldErrors)
      setFormError(inviteRefusal(error) ?? failure.formError)
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

      <Show when={!props.inviteToken}>
        <p class="text-sm text-muted-foreground">
          Esta mesa é por convite: peça o link a quem administra.
        </p>
      </Show>

      <Show when={formError()}>
        {(message) => <p class="text-sm text-destructive">{message()}</p>}
      </Show>

      <Button type="submit" class="w-full" disabled={submitting()}>
        {submitting() ? 'Criando…' : 'Criar conta'}
      </Button>
    </form>
  )
}

/**
 * The server refuses a missing, spent or expired link with the same 403 — on
 * purpose, since telling an anonymous caller which one only helps someone
 * probing tokens. Its message is English (the whole API layer is); the player
 * reads this instead.
 */
function inviteRefusal(error: unknown): string | null {
  if (!(error instanceof ApiError) || error.status !== 403) return null
  return 'Convite inválido ou expirado. Peça um link novo a quem administra a mesa.'
}
