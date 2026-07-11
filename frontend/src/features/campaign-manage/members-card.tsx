import { Link } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Trash2, UserPlus } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { Combobox } from '@/shared/ui/combobox'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { ApiError, api } from '@/shared/api/api'
import type {
  CampaignMember,
  CampaignMemberRole,
  Character,
} from '@/shared/api/api'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { charactersQueryOptions } from '@/entities/character/queries'
export function MembersCard({ campaignId }: { campaignId: number }) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))

  return (
    <Card>
      <CardHeader>
        <SectionHeading as="h2">Membros</SectionHeading>
      </CardHeader>
      <CardContent className="space-y-4">
        <AddMemberForm campaignId={campaignId} />
        {members.isLoading && <SkeletonRows count={2} />}
        {members.data?.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhum personagem inscrito ainda.
          </p>
        )}
        <div className="space-y-2">
          {members.data?.map((m) => (
            <MemberRow key={m.id} member={m} campaignId={campaignId} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function AddMemberForm({ campaignId }: { campaignId: number }) {
  const qc = useQueryClient()
  const characters = useQuery(charactersQueryOptions)
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const [characterId, setCharacterId] = useState<string>('')
  const [role, setRole] = useState<CampaignMemberRole>('player')
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () =>
      api.members.add(campaignId, {
        characterId: Number(characterId),
        role,
      }),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: campaignMembersQueryOptions(campaignId).queryKey,
      })
      /* The character just joined — its "Campanhas" tab is stale. */
      qc.invalidateQueries({
        queryKey: ['characters', Number(characterId), 'campaigns'],
      })
      setCharacterId('')
      setError(null)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao adicionar')
    },
  })

  const alreadyIn = new Set(members.data?.map((m) => m.characterId) ?? [])
  const available = (characters.data ?? []).filter(
    (c) => !alreadyIn.has(c.id),
  )

  return (
    <div className="rounded-md border border-dashed p-3">
      <p className="mb-2 text-sm font-medium">Adicionar personagem</p>
      <div className="flex flex-wrap items-end gap-2">
        <div className="min-w-[220px] flex-1">
          <Combobox
            options={available.map((c: Character) => ({
              value: String(c.id),
              label: `${c.name} (Lv ${c.level})`,
            }))}
            value={characterId}
            onChange={setCharacterId}
            placeholder="Selecionar personagem"
            searchPlaceholder="Buscar…"
            emptyMessage="Nenhum personagem disponível."
          />
        </div>
        <div className="w-32">
          <Combobox
            options={[
              { value: 'player', label: 'Jogador' },
              { value: 'gm', label: 'GM' },
            ]}
            value={role}
            onChange={(v) => setRole(v as CampaignMemberRole)}
            placeholder="Papel"
            searchPlaceholder=""
            emptyMessage=""
          />
        </div>
        <Button
          disabled={!characterId || mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          <UserPlus className="mr-1 size-4" />
          {mutation.isPending ? 'Adicionando…' : 'Adicionar'}
        </Button>
      </div>
      {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
    </div>
  )
}

function MemberRow({
  member,
  campaignId,
}: {
  member: CampaignMember
  campaignId: number
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
    <div className="flex items-center justify-between rounded-md border p-2 text-sm transition hover:border-[color:var(--primary)]/50">
      <Link
        to="/characters/$id"
        params={{ id: String(member.characterId) }}
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
      <Button
        variant="ghost"
        size="sm"
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        aria-label={`Remover ${char?.name ?? 'personagem'}`}
      >
        <Trash2 className="size-4" />
      </Button>
    </div>
  )
}
