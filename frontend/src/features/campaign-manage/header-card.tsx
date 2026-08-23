import { useQueryClient } from '@tanstack/solid-query'
import { CalendarClock } from 'lucide-solid'
import { Show, createSignal } from 'solid-js'
import { campaignQueryOptions, campaignsQueryOptions } from '@/entities/campaign/queries'
import { type Campaign, type UpdateCampaignInput, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { CampaignForm } from './campaign-form'
import { SectionTitle } from '@/shared/ui/section-label'

export type CampaignEditFormProps = {
  campaign: Campaign
  onSave: (input: UpdateCampaignInput) => Promise<void>
  onCancel: () => void
}

/**
 * The chronicle's ledger, open for writing. Shares its fields and validation
 * with "abrir nova campanha" via `CampaignForm`; only the framing and the verb
 * differ.
 *
 * @example <CampaignEditForm campaign={c} onSave={save} onCancel={stop} />
 */
export function CampaignEditForm(props: CampaignEditFormProps) {
  return (
    <FramedPanel>
      <SectionTitle contexto="painel" class="text-xl mb-4">
        Editar campanha
      </SectionTitle>
      <CampaignForm
        initial={{
          name: props.campaign.name,
          description: props.campaign.description ?? '',
        }}
        submitLabel="Salvar"
        pendingLabel="Salvando…"
        onSubmit={props.onSave}
        onCancel={props.onCancel}
      />
    </FramedPanel>
  )
}

/** The ledger at rest: title, when it was opened, and the way into editing. */
function CampaignLedger(props: { campaign: Campaign; onEdit: () => void }) {
  return (
    <FramedPanel>
      <div class="flex flex-row items-start justify-between gap-4">
        <div class="space-y-1">
          <SectionTitle contexto="painel" class="text-xl">
            {props.campaign.name}
          </SectionTitle>
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
