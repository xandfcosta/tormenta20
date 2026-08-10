import { useQuery } from '@tanstack/react-query'
import { Crown, ScrollText, Skull, Users } from 'lucide-react'
import type { ReactNode } from 'react'
import type { CampaignMember, Session } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { FramedPanel } from '@/shared/ui/framed-panel'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { InviteButton } from './invite-button'
import { SessionLog } from './session-log'
import { TomeSection } from './tome-section'

/**
 * Visão geral: the chronicle's dashboard — sigils for the party/log at a glance,
 * a party muster preview and the recent-sessions log, each linking into its own
 * section. The live-session CTA lives on the tome header above (ALE-59), so it
 * isn't duplicated here.
 */
export function CampaignOverview({
  campaignId,
  isGm,
  onGoToTab,
}: {
  campaignId: number
  isGm: boolean
  onGoToTab: (tab: 'sessoes' | 'membros') => void
}) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const memberList = members.data ?? []
  const sessionList = sessions.data ?? []

  return (
    <div className="space-y-6">
      <ChronicleSigils members={memberList} sessions={sessionList} />
      <div className="grid gap-5 lg:grid-cols-2">
        <PartyMuster
          members={memberList}
          isLoading={members.isLoading}
          isGm={isGm}
          campaignId={campaignId}
          onSeeAll={() => onGoToTab('membros')}
        />
        <RecentChronicle
          sessions={sessionList}
          isLoading={sessions.isLoading}
          campaignId={campaignId}
          onSeeAll={() => onGoToTab('sessoes')}
        />
      </div>
    </div>
  )
}

/** Three engraved sigils summarizing the chronicle: heroes, sessions, closed. */
function ChronicleSigils({
  members,
  sessions,
}: {
  members: CampaignMember[]
  sessions: Session[]
}) {
  const heroes = members.filter((m) => m.role === 'player').length
  const ended = sessions.filter((s) => s.status === 'ended').length
  return (
    <div className="grid grid-cols-3 gap-3">
      <Sigil icon={<Users className="size-4" />} value={heroes} label="Heróis" />
      <Sigil
        icon={<ScrollText className="size-4" />}
        value={sessions.length}
        label="Sessões"
      />
      <Sigil icon={<Skull className="size-4" />} value={ended} label="Encerradas" />
    </div>
  )
}

function Sigil({
  icon,
  value,
  label,
}: {
  icon: ReactNode
  value: number
  label: string
}) {
  return (
    <div className="flex flex-col items-center gap-1 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-4 text-center">
      <span aria-hidden className="text-grimorio-gold/70">
        {icon}
      </span>
      <span className="font-heading text-3xl leading-none text-grimorio-gold">
        {value}
      </span>
      <span className="text-[10px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
        {label}
      </span>
    </div>
  )
}

/** Party preview: the first heroes as portrait chips, GM crowned. */
function PartyMuster({
  members,
  isLoading,
  isGm,
  campaignId,
  onSeeAll,
}: {
  members: CampaignMember[]
  isLoading: boolean
  isGm: boolean
  campaignId: number
  onSeeAll: () => void
}) {
  const roster = [...members].sort((a, b) =>
    a.role === b.role ? 0 : a.role === 'gm' ? -1 : 1,
  )
  return (
    <FramedPanel>
      <TomeSection
        eyebrow="A Mesa"
        title="Grupo"
        action={isGm && <InviteButton campaignId={campaignId} />}
      >
        {isLoading && <SkeletonRows count={2} />}
        {!isLoading && roster.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhum personagem inscrito ainda.
          </p>
        )}
        {roster.length > 0 && (
          <>
            <div className="flex flex-wrap gap-2">
              {roster.slice(0, 6).map((m) => (
                <MusterChip key={m.id} member={m} />
              ))}
            </div>
            <SeeAll onClick={onSeeAll}>
              Ver {roster.length} {roster.length === 1 ? 'membro' : 'membros'}
            </SeeAll>
          </>
        )}
      </TomeSection>
    </FramedPanel>
  )
}

function MusterChip({ member }: { member: CampaignMember }) {
  const name = member.character?.name ?? `Personagem ${member.characterId}`
  return (
    <span className="flex items-center gap-2 rounded-sm border border-grimorio-iron/60 py-1 pl-1 pr-2.5">
      <CharacterPortrait name={name} size="sm" hue={hueFromName(name)} />
      <span className="flex items-center gap-1 text-sm">
        <span className="max-w-[8rem] truncate font-medium">{name}</span>
        {member.role === 'gm' && (
          <Crown aria-hidden className="size-3.5 text-grimorio-gold" />
        )}
      </span>
    </span>
  )
}

/** Recent-sessions preview — the top few log entries + a link to the full log. */
function RecentChronicle({
  sessions,
  isLoading,
  campaignId,
  onSeeAll,
}: {
  sessions: Session[]
  isLoading: boolean
  campaignId: number
  onSeeAll: () => void
}) {
  return (
    <FramedPanel>
      <TomeSection eyebrow="Crônica" title="Sessões recentes">
        {isLoading && <SkeletonRows count={2} />}
        {!isLoading && sessions.length === 0 && (
          <p className="text-sm text-muted-foreground">Nenhuma sessão ainda.</p>
        )}
        {sessions.length > 0 && (
          <>
            <SessionLog sessions={sessions} campaignId={campaignId} limit={3} />
            <SeeAll onClick={onSeeAll}>Ver todas</SeeAll>
          </>
        )}
      </TomeSection>
    </FramedPanel>
  )
}

/** Gilt link to a section's full page. */
function SeeAll({
  onClick,
  children,
}: {
  onClick: () => void
  children: ReactNode
}) {
  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={onClick}
      className="mt-3 text-grimorio-gold hover:text-grimorio-gold"
    >
      {children} →
    </Button>
  )
}
