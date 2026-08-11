import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { type CampaignFormValues, campaignSchema } from '@/entities/campaign/campaign-schema'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { TextField } from '@/shared/ui/text-field'
import { TextAreaField } from '@/shared/ui/textarea-field'

const BLANK: CampaignFormValues = { name: '', description: '' }

export type CampaignFormProps = {
  /** Starting values; omitted for a blank chronicle. */
  initial?: CampaignFormValues
  submitLabel: string
  pendingLabel: string
  onSubmit: (values: CampaignFormValues) => Promise<void>
  onCancel: () => void
}

/**
 * The chronicle's two fields with the app's validation grammar — the body of
 * both "abrir nova crônica" and "editar campanha". Persisting is the caller's
 * job, so this renders in a test with no query client and no router.
 *
 * @example
 * <CampaignForm submitLabel="Abrir crônica" pendingLabel="Abrindo…"
 *   onSubmit={create} onCancel={back} />
 */
export function CampaignForm(props: CampaignFormProps) {
  const [name, setName] = createSignal(props.initial?.name ?? BLANK.name)
  const [description, setDescription] = createSignal(
    props.initial?.description ?? BLANK.description,
  )
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [pending, setPending] = createSignal(false)

  const showFailure = (failure: unknown) => {
    const submit = toSubmitFailure(failure)
    setFieldErrors(submit.fieldErrors)
    setFormError(submit.formError)
  }

  const handleSubmit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = campaignSchema.safeParse({ name: name(), description: description() })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setPending(true)
    try {
      await props.onSubmit(parsed.data)
    } catch (failure) {
      showFailure(failure)
    } finally {
      setPending(false)
    }
  }

  return (
    <form class="space-y-4" onSubmit={handleSubmit} noValidate>
      <TextField
        name="name"
        label="Nome"
        value={name()}
        onInput={setName}
        errors={fieldErrors().name}
      />
      <TextAreaField
        name="description"
        label="Descrição"
        value={description()}
        onInput={setDescription}
        errors={fieldErrors().description}
      />
      <Show when={formError()}>
        {(message) => <p class="text-sm text-destructive">{message()}</p>}
      </Show>
      <div class="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={() => props.onCancel()}>
          Cancelar
        </Button>
        <Button type="submit" disabled={pending()}>
          {pending() ? props.pendingLabel : props.submitLabel}
        </Button>
      </div>
    </form>
  )
}
