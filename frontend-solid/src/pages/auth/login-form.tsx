import { For, Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { ApiError, type Credentials } from '@/shared/api/api'

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

      <button
        type="submit"
        disabled={submitting()}
        class="h-9 w-full rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground shadow-xs transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50"
      >
        {submitting() ? 'Entrando…' : 'Entrar'}
      </button>
    </form>
  )
}

/** Labelled input + its error list. Replaced by the UI kit's Field in ALE-65. */
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
      <label for={props.name} class="text-sm font-medium">
        {props.label}
      </label>
      <input
        id={props.name}
        name={props.name}
        type={props.type}
        autocomplete={props.autocomplete}
        value={props.value}
        onInput={(e) => props.onInput(e.currentTarget.value)}
        aria-invalid={invalid()}
        class="h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 aria-invalid:border-destructive"
      />
      <For each={props.errors}>
        {(message) => <p class="text-sm text-destructive">{message}</p>}
      </For>
    </div>
  )
}
