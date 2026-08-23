import { Flame } from 'lucide-solid'
import type { Campaign } from '@/shared/api/api'
import { CampaignRulesCard } from './campaign-rules-card'
import { DeleteCampaignButton } from './delete-campaign-button'
import { CampaignHeaderCard } from './header-card'
import { SectionLabel, SectionTitle } from '@/shared/ui/section-label'

/**
 * Config as the tome's settings leaf: the chronicle's ledger (edit) over a
 * sealed danger zone. Light touch — the ledger already carries its own form.
 */
export function ConfigSection(props: { campaign: Campaign }) {
  return (
    <div class="space-y-6">
      <SectionLabel class="font-semibold">
        Ajustes do tomo
      </SectionLabel>
      <CampaignHeaderCard campaign={props.campaign} />
      {/* Entre o cadastro e a zona de perigo: as regras são configuração
          corriqueira, e não destruição (ALE-221). */}
      <CampaignRulesCard
        campaignId={props.campaign.id}
        ignoredRules={props.campaign.ignoredRules ?? []}
      />
      <DangerZone campaign={props.campaign} />
    </div>
  )
}

/** Sealed-in-crimson destructive zone — deleting the chronicle is irreversible. */
function DangerZone(props: { campaign: Campaign }) {
  return (
    <div class="flex flex-wrap items-center justify-between gap-3 rounded-none border border-[color:var(--grimorio-crimson)]/50 bg-[color:var(--grimorio-crimson)]/[0.06] p-4">
      <div class="space-y-1">
        <SectionTitle as="p" contexto="painel" tom="inherit" class="text-sm flex items-center gap-1.5 text-[color:var(--grimorio-crimson-bright)]">
          <Flame aria-hidden="true" class="size-4" />
          Zona de perigo
        </SectionTitle>
        <p class="text-xs text-muted-foreground">
          Excluir a campanha remove todas as sessões e membros. Não pode ser desfeito.
        </p>
      </div>
      <DeleteCampaignButton campaign={props.campaign} />
    </div>
  )
}
