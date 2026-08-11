import { useQuery } from '@tanstack/solid-query'
import { Crown, ScrollText, Skull, Users } from 'lucide-solid'
import { For, type JSX, Show } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import type { CampaignMember, Session } from '@/shared/api/api'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { memberName, sortRoster } from './members-card'
import { SessionLog } from './session-log'
import { TomeSection } from './tome-section'

/**
 * Visão geral: the chronicle's dashboard — sigils for the party/log at a
 * glance, a party muster preview and the recent-sessions log, each linking
 * into its own section. The live-session CTA lives on the tome header above,
 * so it isn't duplicated here.
 */
export function CampaignOverview(props: {
  campaignId: number
  isGm: boolean
  onGoToTab: (tab: 'sessoes' | 'membros') => void
}) {
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))
  const sessions = useQuery(() => campaignSessionsQueryOptions(props.campaignId))
  const memberList = () => members.data ?? []
  const sessionList = () => sessions.data ?? []

  return (
    <div class="space-y-6">
      <ChronicleSigils members={memberList()} sessions={sessionList()} />
      <div class="grid gap-5 lg:grid-cols-2">
        <PartyMuster
          members={memberList()}
          isLoading={members.isLoading}
          onSeeAll={() => props.onGoToTab('membros')}
        />
        <RecentChronicle
          sessions={sessionList()}
          isLoading={sessions.isLoading}
          campaignId={props.campaignId}
          onSeeAll={() => props.onGoToTab('sessoes')}
        />
      </div>
    </div>
  )
}

/** Three engraved sigils summarizing the chronicle: heroes, sessions, closed. */
function ChronicleSigils(props: { members: CampaignMember[]; sessions: Session[] }) {
  const heroes = () => props.members.filter((m) => m.role === 'player').length
  const ended = () => props.sessions.filter((s) => s.status === 'ended').length
  return (
    <div class="grid grid-cols-3 gap-3">
      <Sigil icon={<Users class="size-4" />} value={heroes()} label="Heróis" />
      <Sigil icon={<ScrollText class="size-4" />} value={props.sessions.length} label="Sessões" />
      <Sigil icon={<Skull class="size-4" />} value={ended()} label="Encerradas" />
    </div>
  )
}

function Sigil(props: { icon: JSX.Element; value: number; label: string }) {
  return (
    <div class="flex flex-col items-center gap-1 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-4 text-center">
      <span aria-hidden="true" class="text-grimorio-gold/70">
        {props.icon}
      </span>
      <span class="font-heading text-3xl leading-none text-grimorio-gold">{props.value}</span>
      <span class="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
        {props.label}
      </span>
    </div>
  )
}

/** Party preview: the first heroes as portrait chips, GM crowned. */
function PartyMuster(props: {
  members: CampaignMember[]
  isLoading: boolean
  onSeeAll: () => void
}) {
  const roster = () => sortRoster(props.members)
  return (
    <FramedPanel>
      <TomeSection eyebrow="A Mesa" title="Grupo">
        <Show when={props.isLoading}>
          <SkeletonRows count={2} />
        </Show>
        <Show when={!props.isLoading && roster().length === 0}>
          <p class="text-sm text-muted-foreground">Nenhum personagem inscrito ainda.</p>
        </Show>
        <Show when={roster().length > 0}>
          <div class="flex flex-wrap gap-2">
            <For each={roster().slice(0, 6)}>{(member) => <MusterChip member={member} />}</For>
          </div>
          <SeeAll onClick={props.onSeeAll}>
            Ver {roster().length} {roster().length === 1 ? 'membro' : 'membros'}
          </SeeAll>
        </Show>
      </TomeSection>
    </FramedPanel>
  )
}

function MusterChip(props: { member: CampaignMember }) {
  const name = () => memberName(props.member)
  return (
    <span class="flex items-center gap-2 rounded-sm border border-grimorio-iron/60 py-1 pl-1 pr-2.5">
      <CharacterPortrait name={name()} size="sm" hue={hueFromName(name())} />
      <span class="flex items-center gap-1 text-sm">
        <span class="max-w-[8rem] truncate font-medium">{name()}</span>
        <Show when={props.member.role === 'gm'}>
          <Crown aria-hidden="true" class="size-3.5 text-grimorio-gold" />
        </Show>
      </span>
    </span>
  )
}

/** Recent-sessions preview — the top few log entries + a link to the full log. */
function RecentChronicle(props: {
  sessions: Session[]
  isLoading: boolean
  campaignId: number
  onSeeAll: () => void
}) {
  return (
    <FramedPanel>
      <TomeSection eyebrow="Crônica" title="Sessões recentes">
        <Show when={props.isLoading}>
          <SkeletonRows count={2} />
        </Show>
        <Show when={!props.isLoading && props.sessions.length === 0}>
          <p class="text-sm text-muted-foreground">Nenhuma sessão ainda.</p>
        </Show>
        <Show when={props.sessions.length > 0}>
          <SessionLog sessions={props.sessions} campaignId={props.campaignId} limit={3} />
          <SeeAll onClick={props.onSeeAll}>Ver todas</SeeAll>
        </Show>
      </TomeSection>
    </FramedPanel>
  )
}

/** Gilt link to a section's full page. */
function SeeAll(props: { onClick: () => void; children: JSX.Element }) {
  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => props.onClick()}
      class="mt-3 text-grimorio-gold hover:text-grimorio-gold"
    >
      {props.children} →
    </Button>
  )
}
