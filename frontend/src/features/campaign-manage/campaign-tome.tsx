import { CalendarClock, ChevronRight, Flame } from 'lucide-react'
import type { Campaign, Session } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
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

/** The content region's spatial layout per section: the dashboard and the member
 *  roster are 2-D grids; the session timeline and the config form are 1-D lists.
 *  Drives `data-nav-layout` so arrow movement matches what the eye sees. */
function sectionLayout(tab: CampaignTab): 'grid' | 'column' {
  return tab === 'visao' || tab === 'membros' ? 'grid' : 'column'
}

// Section marker: left-aligned, with a gold edge + raised fill when active so
// the current section reads clearly on the vertical rail (and the strip on
// phones). Overrides the stock shadcn trigger active style.
const railTabClass =
  'justify-start gap-2 rounded-sm px-3 py-2 text-muted-foreground data-[state=active]:bg-[var(--grimorio-panel-raised)] data-[state=active]:text-grimorio-gold data-[state=active]:shadow-none sm:border-l-2 sm:border-transparent sm:data-[state=active]:border-grimorio-gold'

/**
 * The campaign detail as a page INSIDE the tome (ALE-59): a leather cover holds
 * one large page that fills the scene — you've turned to this chronicle's page.
 * An illuminated title (crest + name), the single live-session action, a gilt
 * rule, then the sections as journal entries. Opening zooms in
 * (`grimorio-page-in`); ESC pops back to the book (handled by the page).
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
    <div className="w-full">
      <div className="grimorio-leather grimorio-page-in p-2.5 sm:p-3">
        <div
          data-tome-root
          className="grimorio-frame--stone flex min-h-[calc(100dvh-7rem)] flex-col gap-6 rounded-sm p-6 sm:p-10"
        >
          <TomeHeader
            campaign={campaign}
            campaignId={campaignId}
            isGm={isGm}
            playerCount={playerCount}
            activeSession={activeSession}
            onResume={onResume}
          />
          <GiltRule />
          <Sections
            campaign={campaign}
            campaignId={campaignId}
            isGm={isGm}
            current={current}
            onTab={onTab}
          />
        </div>
      </div>
    </div>
  )
}

/** Illuminated page head: crest + name + party meta, and the single session CTA. */
function TomeHeader({
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
    <header
      data-nav-region="header"
      data-nav-layout="row"
      className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"
    >
      <div className="flex items-center gap-4">
        <span
          aria-hidden
          className="flex size-14 shrink-0 items-center justify-center rounded-md border border-grimorio-iron font-display text-lg text-white/90 sm:size-16"
          style={{ background: campaignEmblemGradient(campaign.name) }}
        >
          {campaignInitials(campaign.name)}
        </span>
        <div className="space-y-1">
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
            {roleLabel(campaign.role)}
          </p>
          <h1 className="font-display text-3xl uppercase leading-tight tracking-wide text-grimorio-gold sm:text-4xl">
            {campaign.name}
          </h1>
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
            <span>
              {playerCount} {playerCount === 1 ? 'herói' : 'heróis'}
            </span>
            <span aria-hidden>·</span>
            <span className="flex items-center gap-1">
              <CalendarClock className="size-3" />
              Criada em{' '}
              {new Date(campaign.createdAt).toLocaleDateString('pt-BR')}
            </span>
          </p>
        </div>
      </div>
      <SessionAction
        campaignId={campaignId}
        isGm={isGm}
        activeSession={activeSession}
        onResume={onResume}
      />
    </header>
  )
}

/** The one live-session action: Continuar when live, else Nova sessão for a GM. */
function SessionAction({
  campaignId,
  isGm,
  activeSession,
  onResume,
}: {
  campaignId: number
  isGm: boolean
  activeSession: Session | undefined
  onResume: () => void
}) {
  if (activeSession)
    return (
      <div className="shrink-0 sm:text-right">
        <p className="mb-1.5 flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-widest text-[color:var(--hp-full)] sm:justify-end">
          <span className="size-2 animate-pulse rounded-full bg-[color:var(--hp-full)] motion-reduce:animate-none" />
          Sessão {activeSession.sessionNumber} em andamento
        </p>
        <Button size="lg" onClick={onResume}>
          Continuar a sessão
          <ChevronRight aria-hidden className="ml-1 size-4" />
        </Button>
      </div>
    )
  if (isGm)
    return (
      <div className="shrink-0">
        <NewSessionButton campaignId={campaignId} label="Nova sessão" size="lg" />
      </div>
    )
  return (
    <p className="shrink-0 text-xs italic text-muted-foreground">
      Nenhuma sessão ativa.
    </p>
  )
}

/** Gilt rule — an illuminated divider under the title. */
function GiltRule() {
  return (
    <div
      aria-hidden
      className="h-px w-full bg-gradient-to-r from-transparent via-grimorio-gold/40 to-transparent"
    />
  )
}

/** The sections written on the page, behind marker tabs. */
function Sections({
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
    <Tabs
      value={current}
      onValueChange={(v) => onTab(v as CampaignTab)}
      orientation="vertical"
      className="flex min-h-0 flex-1 flex-col gap-5 sm:flex-row sm:gap-6"
    >
      {/* Section markers down the fore-edge — a vertical index (finger tabs):
          a fixed rail on desktop, a full-width stack on phones. ↑/↓ walk
          between them — see the page's keyboard handler. Radix forces the list
          to flex-col under orientation=vertical, so we don't fight it. */}
      <div className="flex shrink-0 flex-col gap-2 sm:w-44">
        <TabsList
          data-nav-region="rail"
          data-nav-layout="column"
          data-nav-edge-right="content"
          data-nav-edge-up="header"
          className="flex h-auto flex-col gap-1 bg-transparent p-0 sm:items-stretch"
        >
          <TabsTrigger value="visao" className={railTabClass}>
            Visão geral
          </TabsTrigger>
          <TabsTrigger value="sessoes" className={railTabClass}>
            Sessões
          </TabsTrigger>
          <TabsTrigger value="membros" className={railTabClass}>
            Membros
          </TabsTrigger>
          {isGm && (
            <TabsTrigger value="config" className={railTabClass}>
              Config
            </TabsTrigger>
          )}
        </TabsList>
        <p className="hidden px-2 text-[10px] uppercase tracking-widest text-muted-foreground xl:block">
          ↑↓←→ navegar · PgUp/PgDn seção · Esc voltar
        </p>
      </div>

      <div
        data-nav-region="content"
        data-nav-layout={sectionLayout(current)}
        data-nav-edge-left="rail"
        data-nav-edge-up="header"
        className="min-w-0 flex-1"
        data-tome-content
      >
      <TabsContent value="visao">
        <CampaignOverview campaignId={campaignId} isGm={isGm} onGoToTab={onTab} />
      </TabsContent>
      <TabsContent value="sessoes">
        <SessionsCard campaignId={campaignId} isGm={isGm} />
      </TabsContent>
      <TabsContent value="membros">
        <MembersCard campaignId={campaignId} isGm={isGm} />
      </TabsContent>
      {isGm && (
        <TabsContent value="config">
          <ConfigSection campaign={campaign} />
        </TabsContent>
      )}
      </div>
    </Tabs>
  )
}

/** Config as the tome's settings leaf: the chronicle's ledger (edit) over a
 *  sealed danger zone. Light touch — the header card already carries its form. */
function ConfigSection({ campaign }: { campaign: Campaign }) {
  return (
    <div className="space-y-6">
      <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-muted-foreground">
        Ajustes do tomo
      </p>
      <CampaignHeaderCard campaign={campaign} />
      <DangerZone campaign={campaign} />
    </div>
  )
}

/** Sealed-in-crimson destructive zone — deleting the chronicle is irreversible. */
function DangerZone({ campaign }: { campaign: Campaign }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-sm border border-[color:var(--grimorio-crimson)]/50 bg-[color:var(--grimorio-crimson)]/[0.06] p-4">
      <div className="space-y-1">
        <p className="flex items-center gap-1.5 font-heading text-sm uppercase tracking-wide text-[color:var(--grimorio-crimson-bright)]">
          <Flame aria-hidden className="size-4" />
          Zona de perigo
        </p>
        <p className="text-xs text-muted-foreground">
          Excluir a campanha remove todas as sessões e membros. Não pode ser
          desfeito.
        </p>
      </div>
      <DeleteCampaignButton campaign={campaign} />
    </div>
  )
}
