import { Flame } from 'lucide-solid'
import type { Campaign } from '@/shared/api/api'
import { DeleteCampaignButton } from './delete-campaign-button'
import { CampaignHeaderCard } from './header-card'

/**
 * Config as the tome's settings leaf: the chronicle's ledger (edit) over a
 * sealed danger zone. Light touch — the ledger already carries its own form.
 */
export function ConfigSection(props: { campaign: Campaign }) {
  return (
    <div class="space-y-6">
      <p class="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
        Ajustes do tomo
      </p>
      <CampaignHeaderCard campaign={props.campaign} />
      <DangerZone campaign={props.campaign} />
    </div>
  )
}

/** Sealed-in-crimson destructive zone — deleting the chronicle is irreversible. */
function DangerZone(props: { campaign: Campaign }) {
  return (
    <div class="flex flex-wrap items-center justify-between gap-3 rounded-none border border-[color:var(--grimorio-crimson)]/50 bg-[color:var(--grimorio-crimson)]/[0.06] p-4">
      <div class="space-y-1">
        <p class="flex items-center gap-1.5 font-heading text-sm uppercase tracking-wide text-[color:var(--grimorio-crimson-bright)]">
          <Flame aria-hidden="true" class="size-4" />
          Zona de perigo
        </p>
        <p class="text-xs text-muted-foreground">
          Excluir a campanha remove todas as sessões e membros. Não pode ser desfeito.
        </p>
      </div>
      <DeleteCampaignButton campaign={props.campaign} />
    </div>
  )
}
