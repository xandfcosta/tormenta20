import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Crown, Trash2, Users } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { cn } from '@/shared/lib/utils'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { api } from '@/shared/api/api'
import type { CampaignMember } from '@/shared/api/api'
import { classLevelLine } from '@/entities/character/class-line'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { InviteButton } from './invite-button'
import { TomeSection } from './tome-section'

/**
 * Membros section as a PARTY ROSTER — every hero as a portrait plate (the GM
 * crowned), laid out as a grid that grows columns with the tome page. Players
 * are never added here; the GM shares an invite link (InviteButton) and each
 * player joins with a character of their own.
 */
export function MembersCard({
  campaignId,
  isGm,
}: {
  campaignId: number
  isGm: boolean
}) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  // GM first (the crest leads the muster), then the rest in arrival order.
  const roster = [...(members.data ?? [])].sort((a, b) =>
    a.role === b.role ? 0 : a.role === 'gm' ? -1 : 1,
  )

  return (
    <TomeSection
      eyebrow="A Mesa"
      title="Heróis"
      action={isGm && <InviteButton campaignId={campaignId} />}
    >
      {isGm && (
        <p className="text-sm text-muted-foreground">
          Jogadores entram pelo link de convite com um personagem próprio.
        </p>
      )}
      {members.isLoading && <SkeletonRows count={2} />}
      {!members.isLoading && roster.length === 0 && <EmptyMuster />}
      {roster.length > 0 && (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {roster.map((m) => (
            <MemberPlate
              key={m.id}
              member={m}
              campaignId={campaignId}
              canRemove={isGm}
            />
          ))}
        </div>
      )}
    </TomeSection>
  )
}

/** No members yet — an empty muster awaiting the first hero. */
function EmptyMuster() {
  return (
    <div className="flex flex-col items-center gap-2 rounded-sm border border-dashed border-grimorio-iron px-4 py-10 text-center">
      <Users aria-hidden className="size-6 text-muted-foreground" />
      <p className="text-sm text-muted-foreground">
        Nenhum personagem inscrito ainda.
      </p>
    </div>
  )
}

function MemberPlate({
  member,
  campaignId,
  canRemove,
}: {
  member: CampaignMember
  campaignId: number
  canRemove: boolean
}) {
  const remove = useRemoveMember(member, campaignId)
  const char = member.character
  const name = char?.name ?? `Personagem ${member.characterId}`
  const isGm = member.role === 'gm'

  return (
    <div
      className={cn(
        'group relative flex gap-3 rounded-sm border bg-[var(--grimorio-panel)] p-3 transition-colors',
        isGm
          ? 'border-grimorio-gold/50 hover:border-grimorio-gold'
          : 'border-grimorio-iron hover:border-grimorio-gold',
      )}
    >
      <Link
        to="/characters/$id"
        params={{ id: member.characterId }}
        className="flex min-w-0 flex-1 items-center gap-3"
      >
        <CharacterPortrait name={name} size="sm" hue={hueFromName(name)} />
        <div className="min-w-0 space-y-1">
          <p className="flex items-center gap-1.5 truncate font-medium text-foreground">
            <span className="truncate">{name}</span>
            {isGm && (
              <Crown aria-hidden className="size-3.5 shrink-0 text-grimorio-gold" />
            )}
          </p>
          <p className="truncate text-xs text-muted-foreground">
            {char ? classLevelLine(char.classes) || `Nível ${char.level}` : '—'}
          </p>
          <RolePill isGm={isGm} />
        </div>
      </Link>
      {canRemove && (
        <Button
          variant="ghost"
          size="icon"
          data-nav-skip
          className="absolute right-1.5 top-1.5 size-7 opacity-0 transition-opacity focus-visible:opacity-100 group-hover:opacity-100"
          onClick={() => remove.mutate()}
          disabled={remove.isPending}
          aria-label={`Remover ${name}`}
        >
          <Trash2 className="size-4" />
        </Button>
      )}
    </div>
  )
}

/** Stance chip: the GM wears a gilt crest, players a plain iron tag. */
function RolePill({ isGm }: { isGm: boolean }) {
  return (
    <span
      className={cn(
        'inline-flex rounded-sm border px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider',
        isGm
          ? 'border-grimorio-gold/60 text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground',
      )}
    >
      {isGm ? 'Mestre' : 'Jogador'}
    </span>
  )
}

/** Drops a member and clears the character's now-stale "Campanhas" list. */
function useRemoveMember(member: CampaignMember, campaignId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.members.remove(campaignId, member.id),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: campaignMembersQueryOptions(campaignId).queryKey,
      })
      qc.invalidateQueries({
        queryKey: ['characters', member.characterId, 'campaigns'],
      })
    },
  })
}
