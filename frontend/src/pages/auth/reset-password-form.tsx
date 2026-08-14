import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { ApiError } from '@/shared/api/api'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { TextField } from '@/shared/ui/text-field'

const resetSchema = z
  .object({
    password: z.string().min(8, 'A senha precisa ter ao menos 8 caracteres'),
    confirm: z.string().min(1, 'Confirme sua senha'),
  })
  .refine((v) => v.password === v.confirm, {
    path: ['confirm'],
    message: 'As senhas não conferem',
  })

/**
 * Escolher a nova senha (ALE-120). Mesmo contrato do {@link RegisterForm}: a
 * validação e a apresentação aqui, a chamada injetada.
 *
 * O `confirm` não sai deste componente; ele existe só para pegar o typo antes
 * de uma senha que o jogador não consegue reproduzir virar a senha da conta —
 * e aqui isso é pior que no registro, porque o link já foi gasto.
 *
 * @example <ResetPasswordForm onSubmit={(senha) => api.auth.resetPassword({ token, password: senha })} />
 */
export function ResetPasswordForm(props: { onSubmit: (password: string) => Promise<void> }) {
  const [password, setPassword] = createSignal('')
  const [confirm, setConfirm] = createSignal('')
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [submitting, setSubmitting] = createSignal(false)

  const handleSubmit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = resetSchema.safeParse({ password: password(), confirm: confirm() })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setSubmitting(true)
    try {
      await props.onSubmit(parsed.data.password)
    } catch (error) {
      const failure = toSubmitFailure(error)
      setFieldErrors(failure.fieldErrors)
      setFormError(linkRefusal(error) ?? failure.formError)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form class="space-y-4" onSubmit={handleSubmit} noValidate>
      <TextField
        name="password"
        label="Nova senha"
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
        {submitting() ? 'Salvando…' : 'Salvar nova senha'}
      </Button>
    </form>
  )
}

/**
 * O servidor recusa link inválido, gasto e vencido com o mesmo 403 — dizer qual
 * dos três só ajudaria quem estivesse sondando tokens. A mensagem dele é em
 * inglês (a camada de API inteira é); o jogador lê esta.
 */
function linkRefusal(error: unknown): string | null {
  if (!(error instanceof ApiError) || error.status !== 403) return null
  return 'Este link não vale mais. Peça outro a quem administra a mesa.'
}
