import { For, Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { ApiError, type Credentials } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { Label } from '@/shared/ui/label'

const loginSchema = z.object({
  email: z.email('E-mail inválido'),
  password: z.string().min(1, 'Informe sua senha'),
})

type FieldName = keyof Credentials
type FieldErrors = Partial<Record<FieldName, string[]>>

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
      setFieldErrors(z.flattenError(parsed.error).fieldErrors)
      return
    }
    setFieldErrors({})
    setSubmitting(true)
    try {
      await props.onSubmit(parsed.data)
    } catch (error) {
      reportFailure(error)
    } finally {
      setSubmitting(false)
    }
  }

  /** Server-side rejections: per-field when the backend says which, else global. */
  const reportFailure = (error: unknown) => {
    if (!(error instanceof ApiError)) {
      setFormError('Erro inesperado. Tente novamente.')
      return
    }
    setFieldErrors(error.fieldErrors)
    if (Object.keys(error.fieldErrors).length === 0) setFormError(error.message)
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

/** Labelled input + its error list, on the kit's Label/Input primitives. */
function TextField(props: {
  name: FieldName
  label: string
  type: 'email' | 'password'
  autocomplete: 'email' | 'current-password'
  value: string
  onInput: (value: string) => void
  errors: string[] | undefined
}) {
  const invalid = () => (props.errors?.length ?? 0) > 0
  return (
    <div class="space-y-2">
      <Label for={props.name}>{props.label}</Label>
      <Input
        id={props.name}
        name={props.name}
        type={props.type}
        autocomplete={props.autocomplete}
        value={props.value}
        onInput={(e) => props.onInput(e.currentTarget.value)}
        aria-invalid={invalid()}
      />
      <For each={props.errors}>{(message) => <p class="text-sm text-destructive">{message}</p>}</For>
    </div>
  )
}
