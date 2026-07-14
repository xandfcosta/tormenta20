import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Trash2 } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { api } from '@/shared/api/api'
import type { CampaignMember } from '@/shared/api/api'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { InviteButton } from './invite-button'

/**
 * Members list. Players are never added directly — the GM shares an invite
 * link (InviteButton) and each player joins with a character of their own.
 * So this card only lists members (+ GM removal) and surfaces the invite.
 */
export function MembersCard({
  campaignId,
  isGm,
}: {
  campaignId: number
  isGm: boolean
}) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <SectionHeading as="h2">Membros</SectionHeading>
        {isGm && <InviteButton campaignId={campaignId} />}
      </CardHeader>
      <CardContent className="space-y-4">
        {isGm && (
          <p className="text-sm text-muted-foreground">
            Jogadores entram pelo link de convite com um personagem próprio.
          </p>
        )}
        {members.isLoading && <SkeletonRows count={2} />}
        {members.data?.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhum personagem inscrito ainda.
          </p>
        )}
        <div className="space-y-2">
          {members.data?.map((m) => (
            <MemberRow
              key={m.id}
              member={m}
              campaignId={campaignId}
              canRemove={isGm}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function MemberRow({
  member,
  campaignId,
  canRemove,
}: {
  member: CampaignMember
  campaignId: number
  canRemove: boolean
}) {
  const qc = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => api.members.remove(campaignId, member.id),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: campaignMembersQueryOptions(campaignId).queryKey,
      })
      /* The character no longer belongs to this campaign — its
       * "Campanhas" tab still lists the stale row. */
      qc.invalidateQueries({
        queryKey: ['characters', member.characterId, 'campaigns'],
      })
    },
  })

  const char = member.character
  const classes =
    char?.classes.map((c) => `${c.className} ${c.level}`).join(' / ') ?? '—'

  return (
    <div className="flex items-center justify-between rounded-md border p-2 text-sm transition-colors hover:border-primary">
      <Link
        to="/characters/$id"
        params={{ id: member.characterId }}
        className="flex-1"
      >
        <p className="flex flex-wrap items-center gap-1 font-medium">
          <span>{char?.name ?? `Character ${member.characterId}`}</span>
          <Badge variant="secondary">Lv {char?.level ?? '?'}</Badge>
          <Badge variant={member.role === 'gm' ? 'default' : 'outline'}>
            {member.role === 'gm' ? 'GM' : 'Jogador'}
          </Badge>
        </p>
        <p className="text-xs text-muted-foreground">{classes}</p>
      </Link>
      {canRemove && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
          aria-label={`Remover ${char?.name ?? 'personagem'}`}
        >
          <Trash2 className="size-4" />
        </Button>
      )}
    </div>
  )
}
