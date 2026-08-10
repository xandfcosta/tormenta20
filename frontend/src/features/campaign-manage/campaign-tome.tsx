import { CalendarClock, ChevronRight } from 'lucide-react'
import type { Campaign, Session } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import {
  campaignEmblemGradient,
  campaignInitials,
  roleLabel,
} from '@/entities/campaign/emblem'
import { CampaignHeaderCard } from './header-card'
import { CampaignOverview } from './campaign-overview'
import { DeleteCampaignButton } from './delete-campaign-button'
import { MembersCard } from './members-card'
import { NewSessionButton } from './new-session-button'
import { SessionsCard } from './sessions-card'

export type CampaignTab = 'visao' | 'sessoes' | 'membros' | 'config'

/**
 * The campaign detail as an open tome (ALE-59), continuing the Crônicas book
 * language. Left leaf = the chronicle's identity (hue emblem, name, role, party,
 * "Continuar a sessão" when live); right leaf = its sections behind marker tabs.
 * The tome zooms in and a leaf turns on mount (`grimorio-tome-open` / `-leaf`),
 * so opening a chronicle reads as flipping to its page. On phones the leaves
 * stack (identity over sections).
 */
export function CampaignTome({
  campaign,
  campaignId,
  isGm,
  activeSession,
  playerCount,
  current,
  onTab,
  onResume,
}: {
  campaign: Campaign
  campaignId: number
  isGm: boolean
  activeSession: Session | undefined
  playerCount: number
  current: CampaignTab
  onTab: (tab: CampaignTab) => void
  onResume: () => void
}) {
  return (
    <div className="mx-auto flex w-full max-w-5xl">
      {/* ONE page (a single framed panel — no book spine to cut the content):
          a hue identity sidebar beside the sections, on a continuous surface. */}
      <div className="grimorio-page-in grimorio-frame grimorio-frame--stone flex w-full flex-col overflow-hidden sm:flex-row sm:items-stretch">
        <IdentityLeaf
          campaign={campaign}
          campaignId={campaignId}
          isGm={isGm}
          playerCount={playerCount}
          activeSession={activeSession}
          onResume={onResume}
        />
        <SectionsLeaf
          campaign={campaign}
          campaignId={campaignId}
          isGm={isGm}
          current={current}
          onTab={onTab}
        />
      </div>
    </div>
  )
}

/** Left leaf — the chronicle's cover: hue emblem, identity, and the single
 *  live-session action (Continuar when live, else Nova sessão for a GM). */
function IdentityLeaf({
  campaign,
  campaignId,
  isGm,
  playerCount,
  activeSession,
  onResume,
}: {
  campaign: Campaign
  campaignId: number
  isGm: boolean
  playerCount: number
  activeSession: Session | undefined
  onResume: () => void
}) {
  return (
    <div
      className="relative flex shrink-0 flex-col justify-between gap-6 overflow-hidden p-6 text-white sm:w-[38%] sm:p-8"
      style={{ background: campaignEmblemGradient(campaign.name) }}
    >
      <span
        aria-hidden
        className="pointer-events-none absolute -right-4 -top-8 select-none font-display text-[8rem] leading-none text-white/10 sm:text-[10rem]"
      >
        {campaignInitials(campaign.name)}
      </span>
      <div className="relative space-y-2">
        <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-white/70">
          {roleLabel(campaign.role)}
        </p>
        <h1 className="font-display text-3xl uppercase leading-tight tracking-wide sm:text-4xl">
          {campaign.name}
        </h1>
        <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-white/80">
          <span>
            {playerCount} {playerCount === 1 ? 'herói' : 'heróis'}
          </span>
          <span aria-hidden>·</span>
          <span className="flex items-center gap-1">
            <CalendarClock className="size-3" />
            Criada em {new Date(campaign.createdAt).toLocaleDateString('pt-BR')}
          </span>
        </p>
      </div>
      <div className="relative">
        {activeSession ? (
          <>
            <p className="mb-2 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-widest">
              <span className="size-2 animate-pulse rounded-full bg-[color:var(--hp-full)] motion-reduce:animate-none" />
              Sessão {activeSession.sessionNumber} em andamento
            </p>
            <Button size="lg" onClick={onResume} className="w-full sm:w-auto">
              Continuar a sessão
              <ChevronRight aria-hidden className="ml-1 size-4" />
            </Button>
          </>
        ) : (
          <>
            <p className="mb-2 text-[11px] font-semibold uppercase tracking-widest text-white/70">
              Nenhuma sessão ativa
            </p>
            {isGm && (
              <NewSessionButton
                campaignId={campaignId}
                label="Nova sessão"
                size="lg"
              />
            )}
          </>
        )}
      </div>
    </div>
  )
}

/** Right leaf — the sections behind marker tabs, on a stone page (the existing
 *  dark section cards fit). A leaf turns over it once on open. */
function SectionsLeaf({
  campaign,
  campaignId,
  isGm,
  current,
  onTab,
}: {
  campaign: Campaign
  campaignId: number
  isGm: boolean
  current: CampaignTab
  onTab: (tab: CampaignTab) => void
}) {
  return (
    <div className="relative flex flex-1 flex-col gap-4 p-5 sm:p-7">
      <Tabs value={current} onValueChange={(v) => onTab(v as CampaignTab)}>
        <TabsList className="max-w-full self-start overflow-x-auto [&>button]:shrink-0">
          <TabsTrigger value="visao">Visão geral</TabsTrigger>
          <TabsTrigger value="sessoes">Sessões</TabsTrigger>
          <TabsTrigger value="membros">Membros</TabsTrigger>
          {isGm && <TabsTrigger value="config">Config</TabsTrigger>}
        </TabsList>

        <TabsContent value="visao">
          <CampaignOverview
            campaignId={campaignId}
            isGm={isGm}
            onGoToTab={onTab}
          />
        </TabsContent>
        <TabsContent value="sessoes">
          <SessionsCard campaignId={campaignId} isGm={isGm} />
        </TabsContent>
        <TabsContent value="membros">
          <MembersCard campaignId={campaignId} isGm={isGm} />
        </TabsContent>
        {isGm && (
          <TabsContent value="config" className="space-y-6">
            <CampaignHeaderCard campaign={campaign} />
            <Card className="border-destructive/40">
              <CardContent className="flex flex-wrap items-center justify-between gap-3 pt-6">
                <div>
                  <p className="font-medium">Excluir campanha</p>
                  <p className="text-xs text-muted-foreground">
                    Remove todas as sessões e membros. Não pode ser desfeito.
                  </p>
                </div>
                <DeleteCampaignButton campaign={campaign} />
              </CardContent>
            </Card>
          </TabsContent>
        )}
      </Tabs>
    </div>
  )
}
