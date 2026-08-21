import { initials } from '@/shared/lib/initials'
import { CalendarClock, ChevronRight } from 'lucide-solid'
import { Show } from 'solid-js'
import {campaignEmblemGradient, roleLabel} from '@/entities/campaign/emblem'
import type { Campaign, Session } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { TomePage } from '@/shared/ui/tome-page'
import { CampaignOverview } from './campaign-overview'
import { ConfigSection } from './config-section'
import { MembersCard } from './members-card'
import { SessionsCard } from './sessions-card'
import { FieldLabel, SectionLabel } from '@/shared/ui/section-label'

export type CampaignTab = 'visao' | 'sessoes' | 'membros' | 'config'

const ALL_TABS: readonly CampaignTab[] = ['visao', 'sessoes', 'membros', 'config']

export function isCampaignTab(value: string | undefined): value is CampaignTab {
  return ALL_TABS.includes(value as CampaignTab)
}

/**
 * The sections a caller can cycle through with the bumpers, in rail order.
 * Config is the GM's leaf: a player's bumpers must not stop on a section that
 * isn't on their rail (ALE-79).
 *
 * @example campaignTabs(false) // ['visao', 'sessoes', 'membros']
 */
export function campaignTabs(isGm: boolean): readonly CampaignTab[] {
  return isGm ? ALL_TABS : ALL_TABS.filter((tab) => tab !== 'config')
}

/**
 * The content region's spatial layout per section: the dashboard and the member
 * roster are 2-D grids; the session timeline and the config form are 1-D lists.
 * Drives `data-nav-layout` so arrow movement matches what the eye sees.
 */
export function sectionLayout(tab: CampaignTab): 'grid' | 'column' {
  return tab === 'visao' || tab === 'membros' ? 'grid' : 'column'
}

// Section marker: left-aligned, with a gold edge + raised fill when active so
// the current section reads clearly on the vertical rail. Kobalte marks the
// active trigger `data-selected` (Radix used data-[state=active]).
const railTabClass =
  'justify-start gap-2 rounded-none px-3 py-2 text-muted-foreground data-[selected]:bg-grimorio-panel-raised data-[selected]:text-grimorio-gold data-[selected]:shadow-none sm:border-l-2 sm:border-transparent sm:data-[selected]:border-grimorio-gold'

export type CampaignTomeProps = {
  campaign: Campaign
  campaignId: number
  isGm: boolean
  activeSession: Session | undefined
  playerCount: number
  current: CampaignTab
  onTab: (tab: CampaignTab) => void
  onResume: () => void
}

/**
 * The campaign detail as a page INSIDE the tome: a leather cover holds one
 * large page that fills the scene — you've turned to this chronicle's page. An
 * illuminated title (crest + name), the single live-session action, a gilt
 * rule, then the sections as journal entries. Opening zooms in; Esc pops back
 * to the book (handled by the page).
 */
export function CampaignTome(props: CampaignTomeProps) {
  return (
    <TomePage>
      <TomeHeader
        campaign={props.campaign}
        isGm={props.isGm}
        playerCount={props.playerCount}
        activeSession={props.activeSession}
        onResume={props.onResume}
      />
      <GiltRule />
      <Sections
        campaign={props.campaign}
        campaignId={props.campaignId}
        isGm={props.isGm}
        current={props.current}
        onTab={props.onTab}
      />
    </TomePage>
  )
}

/** Illuminated page head: crest + name + party meta, and the session CTA. */
function TomeHeader(props: {
  campaign: Campaign
  isGm: boolean
  playerCount: number
  activeSession: Session | undefined
  onResume: () => void
}) {
  return (
    <header
      data-nav-region="header"
      data-nav-layout="row"
      class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"
    >
      <div class="flex items-center gap-4">
        <span
          aria-hidden="true"
          class="flex size-14 shrink-0 items-center justify-center rounded-sm border border-grimorio-iron font-display text-lg text-white/90 sm:size-16"
          style={{ background: campaignEmblemGradient(props.campaign.name) }}
        >
          {initials(props.campaign.name)}
        </span>
        <div class="space-y-1">
          <SectionLabel class="font-semibold">
            {roleLabel(props.campaign.role, props.campaign.ownerName)}
          </SectionLabel>
          <h1 class="font-display text-3xl uppercase leading-tight tracking-wide text-grimorio-gold sm:text-4xl">
            {props.campaign.name}
          </h1>
          <p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
            <span>
              {props.playerCount} {props.playerCount === 1 ? 'herói' : 'heróis'}
            </span>
            <span aria-hidden="true">·</span>
            <span class="flex items-center gap-1">
              <CalendarClock class="size-3" />
              Criada em {new Date(props.campaign.createdAt).toLocaleDateString('pt-BR')}
            </span>
          </p>
        </div>
      </div>
      <SessionAction
        isGm={props.isGm}
        activeSession={props.activeSession}
        onResume={props.onResume}
      />
    </header>
  )
}

/** The live-session action: Continuar when live, else a quiet note. */
function SessionAction(props: {
  isGm: boolean
  activeSession: Session | undefined
  onResume: () => void
}) {
  return (
    <Show
      when={props.activeSession}
      fallback={<p class="shrink-0 text-xs italic text-muted-foreground">Nenhuma sessão ativa.</p>}
    >
      {(session) => (
        <div class="shrink-0 sm:text-right">
          <SectionLabel tom="inherit" class="mb-1.5 flex items-center gap-1.5 font-semibold text-[color:var(--hp-full)] sm:justify-end">
            <span class="size-2 animate-pulse rounded-full bg-[color:var(--hp-full)] motion-reduce:animate-none" />
            Sessão {session().sessionNumber} em andamento
          </SectionLabel>
          <Button size="lg" onClick={() => props.onResume()}>
            Continuar a sessão
            <ChevronRight aria-hidden="true" class="ml-1 size-4" />
          </Button>
        </div>
      )}
    </Show>
  )
}

/** Gilt rule — an illuminated divider under the title. */
function GiltRule() {
  return (
    <div
      aria-hidden="true"
      class="h-px w-full bg-gradient-to-r from-transparent via-grimorio-gold/40 to-transparent"
    />
  )
}

/** The sections written on the page, behind marker tabs. */
function Sections(props: {
  campaign: Campaign
  campaignId: number
  isGm: boolean
  current: CampaignTab
  onTab: (tab: CampaignTab) => void
}) {
  return (
    <Tabs
      value={props.current}
      onChange={(value) => props.onTab(value as CampaignTab)}
      orientation="vertical"
      class="flex min-h-0 flex-1 flex-col gap-5 sm:flex-row sm:gap-6"
    >
      {/* Section markers down the fore-edge — a vertical index (finger tabs):
          a fixed rail on desktop, a full-width stack on phones. */}
      <div class="flex shrink-0 flex-col gap-2 sm:w-44">
        <TabsList
          data-nav-region="rail"
          data-nav-layout="column"
          data-nav-edge-right="content"
          data-nav-edge-up="header"
          class="flex h-auto flex-col gap-1 bg-transparent p-0 sm:items-stretch"
        >
          <TabsTrigger value="visao" class={railTabClass}>
            Visão geral
          </TabsTrigger>
          <TabsTrigger value="sessoes" class={railTabClass}>
            Sessões
          </TabsTrigger>
          <TabsTrigger value="membros" class={railTabClass}>
            Membros
          </TabsTrigger>
          <Show when={props.isGm}>
            <TabsTrigger value="config" class={railTabClass}>
              Config
            </TabsTrigger>
          </Show>
        </TabsList>
        <FieldLabel as="p" class="hidden px-2 xl:block">
          ↑↓←→ navegar · PgUp/PgDn seção · Esc voltar
        </FieldLabel>
      </div>

      <div
        data-nav-region="content"
        data-nav-layout={sectionLayout(props.current)}
        data-nav-edge-left="rail"
        data-nav-edge-up="header"
        data-tome-content
        class="min-w-0 flex-1"
      >
        <TabsContent value="visao">
          <CampaignOverview
            campaignId={props.campaignId}
            isGm={props.isGm}
            onGoToTab={props.onTab}
          />
        </TabsContent>
        <TabsContent value="sessoes">
          <SessionsCard campaignId={props.campaignId} isGm={props.isGm} />
        </TabsContent>
        <TabsContent value="membros">
          <MembersCard campaignId={props.campaignId} isGm={props.isGm} />
        </TabsContent>
        <Show when={props.isGm}>
          <TabsContent value="config">
            <ConfigSection campaign={props.campaign} />
          </TabsContent>
        </Show>
      </div>
    </Tabs>
  )
}
