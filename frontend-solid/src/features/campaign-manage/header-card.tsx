import { useQueryClient } from '@tanstack/solid-query'
import { CalendarClock } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { z } from 'zod'
import { campaignQueryOptions, campaignsQueryOptions } from '@/entities/campaign/queries'
import { type Campaign, type UpdateCampaignInput, api } from '@/shared/api/api'
import { type FieldErrors, toSubmitFailure } from '@/shared/lib/form-errors'
import { Button } from '@/shared/ui/button'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { TextField } from '@/shared/ui/text-field'
import { TextAreaField } from '@/shared/ui/textarea-field'

// Mirrors the create form so edit and create validate alike. `.trim()` runs
// before `.min(1)`, so a name of pure spaces is rejected instead of leaving the
// chronicle untitled in the book.
const campaignEditSchema = z.object({
  name: z.string().trim().min(1, 'Nome é obrigatório').max(120, 'Máximo 120 caracteres'),
  description: z.string().max(2000, 'Máximo 2000 caracteres'),
})

export type CampaignEditFormProps = {
  campaign: Campaign
  onSave: (input: UpdateCampaignInput) => Promise<void>
  onCancel: () => void
}

/**
 * The chronicle's ledger, open for writing: name + description with the app's
 * shared validation grammar. Persisting is the caller's job, so this renders in
 * a test with no query client.
 *
 * @example <CampaignEditForm campaign={c} onSave={save} onCancel={stop} />
 */
export function CampaignEditForm(props: CampaignEditFormProps) {
  const [name, setName] = createSignal(props.campaign.name)
  const [description, setDescription] = createSignal(props.campaign.description ?? '')
  const [fieldErrors, setFieldErrors] = createSignal<FieldErrors>({})
  const [formError, setFormError] = createSignal<string | null>(null)
  const [saving, setSaving] = createSignal(false)

  const showFailure = (failure: unknown) => {
    const submit = toSubmitFailure(failure)
    setFieldErrors(submit.fieldErrors)
    setFormError(submit.formError)
  }

  const handleSubmit = async (event: SubmitEvent) => {
    event.preventDefault()
    setFormError(null)
    const parsed = campaignEditSchema.safeParse({ name: name(), description: description() })
    if (!parsed.success) {
      setFieldErrors(z.flattenError(parsed.error).fieldErrors as FieldErrors)
      return
    }
    setFieldErrors({})
    setSaving(true)
    try {
      await props.onSave(parsed.data)
    } catch (failure) {
      showFailure(failure)
    } finally {
      setSaving(false)
    }
  }

  return (
    <FramedPanel>
      <h2 class="mb-4 font-heading text-xl uppercase tracking-wide text-grimorio-gold">
        Editar campanha
      </h2>
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
          <Button type="submit" disabled={saving()}>
            {saving() ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </form>
    </FramedPanel>
  )
}

/** The ledger at rest: title, when it was opened, and the way into editing. */
function CampaignLedger(props: { campaign: Campaign; onEdit: () => void }) {
  return (
    <FramedPanel>
      <div class="flex flex-row items-start justify-between gap-4">
        <div class="space-y-1">
          <h2 class="font-heading text-xl uppercase tracking-wide text-grimorio-gold">
            {props.campaign.name}
          </h2>
          <p class="flex items-center gap-1 text-xs text-muted-foreground">
            <CalendarClock aria-hidden="true" class="size-3" />
            Criada em {new Date(props.campaign.createdAt).toLocaleDateString('pt-BR')}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => props.onEdit()}>
          Editar
        </Button>
      </div>
      <Show when={props.campaign.description}>
        {(description) => (
          <p class="mt-4 whitespace-pre-line text-sm text-muted-foreground">{description()}</p>
        )}
      </Show>
    </FramedPanel>
  )
}

/** Wires the ledger to the backend, refreshing both the detail and the book. */
export function CampaignHeaderCard(props: { campaign: Campaign }) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = createSignal(false)

  const save = async (input: UpdateCampaignInput) => {
    await api.campaigns.update(props.campaign.id, input)
    await queryClient.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
    await queryClient.invalidateQueries({
      queryKey: campaignQueryOptions(props.campaign.id).queryKey,
    })
    setEditing(false)
  }

  return (
    <Show
      when={editing()}
      fallback={<CampaignLedger campaign={props.campaign} onEdit={() => setEditing(true)} />}
    >
      <CampaignEditForm
        campaign={props.campaign}
        onSave={save}
        onCancel={() => setEditing(false)}
      />
    </Show>
  )
}
