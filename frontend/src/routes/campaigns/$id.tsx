import {
  createFileRoute,
  Link,
  redirect,
  useNavigate,
} from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Combobox } from '@/components/ui/combobox'
import { ApiError, api } from '@/lib/api'
import type {
  Campaign,
  CampaignMember,
  CampaignMemberRole,
  Character,
  Session,
} from '@/lib/api'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
  campaignSessionsQueryOptions,
  campaignsQueryOptions,
  charactersQueryOptions,
  meQueryOptions,
} from '@/lib/queries'

export const Route = createFileRoute('/campaigns/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) => {
    const id = Number(params.id)
    return Promise.all([
      context.queryClient.ensureQueryData(campaignQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignSessionsQueryOptions(id)),
      context.queryClient.ensureQueryData(campaignMembersQueryOptions(id)),
    ])
  },
  component: CampaignDetailPage,
})

function CampaignDetailPage() {
  const { id } = Route.useParams()
  const campaignId = Number(id)
  const campaign = useQuery(campaignQueryOptions(campaignId))

  if (campaign.isLoading) return <p className="p-6">Loading…</p>
  if (campaign.isError)
    return (
      <p className="p-6 text-destructive">
        {(campaign.error as Error).message}
      </p>
    )
  if (!campaign.data) return null

  return (
    <div className="mx-auto h-full max-w-4xl space-y-6 overflow-y-auto p-6">
      <div className="flex items-center justify-between">
        <Link to="/campaigns">
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
        <DeleteCampaignButton campaign={campaign.data} />
      </div>

      <CampaignHeaderCard campaign={campaign.data} />
      <MembersCard campaignId={campaignId} />
      <SessionsCard campaignId={campaignId} />
    </div>
  )
}

// ─── Header + edit ─────────────────────────────────────────────────

function CampaignHeaderCard({ campaign }: { campaign: Campaign }) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [name, setName] = useState(campaign.name)
  const [description, setDescription] = useState(campaign.description ?? '')
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () =>
      api.campaigns.update(campaign.id, {
        name,
        description,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
      qc.invalidateQueries({
        queryKey: campaignQueryOptions(campaign.id).queryKey,
      })
      setEditing(false)
      setError(null)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao salvar')
    },
  })

  if (!editing) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle>{campaign.name}</CardTitle>
            <p className="text-xs text-muted-foreground">
              Criada em{' '}
              {new Date(campaign.createdAt).toLocaleDateString('pt-BR')}
            </p>
          </div>
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            Editar
          </Button>
        </CardHeader>
        {campaign.description && (
          <CardContent className="text-sm">
            <p className="whitespace-pre-line text-muted-foreground">
              {campaign.description}
            </p>
          </CardContent>
        )}
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Editar campanha</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="campaign-name">
            Nome
          </label>
          <Input
            id="campaign-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </div>
        <div className="space-y-1.5">
          <label
            className="text-sm font-medium"
            htmlFor="campaign-description"
          >
            Descrição
          </label>
          <Textarea
            id="campaign-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={6}
          />
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <div className="flex justify-end gap-2">
          <Button
            variant="outline"
            onClick={() => {
              setEditing(false)
              setName(campaign.name)
              setDescription(campaign.description ?? '')
              setError(null)
            }}
          >
            Cancelar
          </Button>
          <Button
            disabled={mutation.isPending || !name.trim()}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? 'Salvando…' : 'Salvar'}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Delete campaign ───────────────────────────────────────────────

function DeleteCampaignButton({ campaign }: { campaign: Campaign }) {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const mutation = useMutation({
    mutationFn: () => api.campaigns.delete(campaign.id),
    onSuccess: async () => {
      qc.invalidateQueries({ queryKey: campaignsQueryOptions.queryKey })
      setOpen(false)
      await navigate({ to: '/campaigns' })
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="destructive" size="sm">
          Excluir campanha
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Excluir "{campaign.name}"?</DialogTitle>
          <DialogDescription>
            Todas as sessões e membros da campanha serão removidos. Esta ação
            não pode ser desfeita.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button
            variant="destructive"
            disabled={mutation.isPending}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? 'Excluindo…' : 'Excluir'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── Members ───────────────────────────────────────────────────────

function MembersCard({ campaignId }: { campaignId: number }) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))

  return (
    <Card>
      <CardHeader>
        <CardTitle>Membros</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <AddMemberForm campaignId={campaignId} />
        {members.isLoading && <p>Loading…</p>}
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
    },
  })

  const char = member.character
  const classes =
    char?.classes.map((c) => `${c.className} ${c.level}`).join(' / ') ?? '—'

  return (
    <div className="flex items-center justify-between rounded-md border p-2 text-sm">
      <div className="flex-1">
        <p className="font-medium">
          {char?.name ?? `Character ${member.characterId}`}{' '}
          <Badge variant="secondary">Lv {char?.level ?? '?'}</Badge>{' '}
          <Badge variant={member.role === 'gm' ? 'default' : 'outline'}>
            {member.role === 'gm' ? 'GM' : 'Jogador'}
          </Badge>
        </p>
        <p className="text-xs text-muted-foreground">{classes}</p>
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
      >
        Remover
      </Button>
    </div>
  )
}

// ─── Sessions ──────────────────────────────────────────────────────

function SessionsCard({ campaignId }: { campaignId: number }) {
  const qc = useQueryClient()
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const navigate = useNavigate()

  const nextNumber =
    (sessions.data?.reduce((max, s) => Math.max(max, s.sessionNumber), 0) ??
      0) + 1

  const mutation = useMutation({
    mutationFn: () =>
      api.sessions.create(campaignId, { sessionNumber: nextNumber }),
    onSuccess: async (created) => {
      qc.invalidateQueries({
        queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
      })
      await navigate({
        to: '/campaigns/$id/sessions/$sid',
        params: { id: String(campaignId), sid: String(created.id) },
      })
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Sessões</CardTitle>
        <Button
          size="sm"
          disabled={mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          {mutation.isPending ? 'Criando…' : `+ Sessão ${nextNumber}`}
        </Button>
      </CardHeader>
      <CardContent className="space-y-2">
        {sessions.isLoading && <p>Loading…</p>}
        {sessions.data?.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhuma sessão ainda.
          </p>
        )}
        {sessions.data?.map((s) => (
          <SessionRow key={s.id} session={s} campaignId={campaignId} />
        ))}
      </CardContent>
    </Card>
  )
}

function SessionRow({
  session,
  campaignId,
}: {
  session: Session
  campaignId: number
}) {
  const badgeVariant =
    session.status === 'active'
      ? 'default'
      : session.status === 'ended'
        ? 'secondary'
        : 'outline'
  return (
    <Link
      to="/campaigns/$id/sessions/$sid"
      params={{ id: String(campaignId), sid: String(session.id) }}
    >
      <div className="flex items-center justify-between rounded-md border p-2 text-sm transition hover:border-primary/40">
        <div>
          <p className="font-medium">
            Sessão {session.sessionNumber}{' '}
            {session.title && (
              <span className="text-muted-foreground">— {session.title}</span>
            )}
          </p>
          <p className="text-xs text-muted-foreground">
            {new Date(session.createdAt).toLocaleDateString('pt-BR')}
          </p>
        </div>
        <Badge variant={badgeVariant}>
          {session.status === 'planned'
            ? 'Planejada'
            : session.status === 'active'
              ? 'Ativa'
              : 'Encerrada'}
        </Badge>
      </div>
    </Link>
  )
}
