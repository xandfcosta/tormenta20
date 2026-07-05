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
import { NumberInput } from '@/components/ui/number-input'
import { Textarea } from '@/components/ui/textarea'
import { ApiError, api } from '@/lib/api'
import type { Session, SessionStatus } from '@/lib/api'
import {
  campaignSessionQueryOptions,
  campaignSessionsQueryOptions,
  meQueryOptions,
} from '@/lib/queries'
import { useSessionSocket, type InitiativeEntry } from '@/lib/realtime'

export const Route = createFileRoute('/campaigns/$id/sessions/$sid')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      campaignSessionQueryOptions(Number(params.id), Number(params.sid)),
    ),
  component: SessionDetailPage,
})

function SessionDetailPage() {
  const { id, sid } = Route.useParams()
  const campaignId = Number(id)
  const sessionId = Number(sid)
  const session = useQuery(
    campaignSessionQueryOptions(campaignId, sessionId),
  )

  if (session.isLoading) return <p className="p-6">Loading…</p>
  if (session.isError)
    return (
      <p className="p-6 text-destructive">
        {(session.error as Error).message}
      </p>
    )
  if (!session.data) return null

  return (
    <div className="mx-auto h-full max-w-3xl space-y-6 overflow-y-auto p-6">
      <div className="flex items-center justify-between">
        <Link to="/campaigns/$id" params={{ id }}>
          <Button variant="outline" size="sm">
            ← Voltar para a campanha
          </Button>
        </Link>
        <DeleteSessionButton
          campaignId={campaignId}
          sessionId={sessionId}
          sessionNumber={session.data.sessionNumber}
        />
      </div>

      <HeaderCard
        campaignId={campaignId}
        session={session.data}
      />
      <InitiativeCard campaignId={campaignId} sessionId={sessionId} />
      <NotesCard campaignId={campaignId} session={session.data} />
    </div>
  )
}

// ─── Initiative (realtime) ──────────────────────────────────────

function InitiativeCard({
  campaignId,
  sessionId,
}: {
  campaignId: number
  sessionId: number
}) {
  const rt = useSessionSocket(campaignId, sessionId)
  const [addLabel, setAddLabel] = useState('')
  const [addInit, setAddInit] = useState(10)
  const [addType, setAddType] = useState<'character' | 'npc'>('npc')

  const submitAdd = () => {
    const label = addLabel.trim()
    if (!label) return
    rt.addEntry({
      label,
      initiative: addInit,
      type: addType,
    })
    setAddLabel('')
    setAddInit(10)
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <div className="space-y-1">
          <CardTitle>Iniciativa</CardTitle>
          <div className="flex gap-2 text-xs text-muted-foreground">
            <Badge variant={rt.isConnected ? 'default' : 'secondary'}>
              {rt.isConnected ? 'Conectado' : 'Desconectado'}
            </Badge>
            <span>Rodada {rt.state.round}</span>
          </div>
        </div>
        <div className="flex gap-2">
          <Button size="sm" onClick={rt.nextTurn} disabled={!rt.isConnected}>
            Próximo turno
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={rt.resetInitiative}
            disabled={!rt.isConnected}
          >
            Reset
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {rt.error && (
          <p className="text-sm text-destructive">
            Erro realtime: {rt.error}
          </p>
        )}

        {rt.state.initiative.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Sem combatentes ainda. Adicione abaixo.
          </p>
        )}

        <div className="space-y-1">
          {rt.state.initiative.map((entry, idx) => (
            <InitiativeRow
              key={entry.id}
              entry={entry}
              onTurn={idx === rt.state.turnIndex}
              onDeltaHp={(delta) =>
                rt.deltaVitals(entry.id, { hpDelta: delta })
              }
              onRemove={() => rt.removeEntry(entry.id)}
            />
          ))}
        </div>

        <div className="mt-3 flex flex-wrap items-end gap-2 rounded-md border border-dashed p-3">
          <div className="min-w-[160px] flex-1">
            <label className="text-xs font-medium" htmlFor="add-label">
              Nome
            </label>
            <Input
              id="add-label"
              value={addLabel}
              onChange={(e) => setAddLabel(e.target.value)}
              placeholder="Goblin salteador…"
            />
          </div>
          <div className="w-24">
            <label className="text-xs font-medium" htmlFor="add-init">
              Iniciativa
            </label>
            <NumberInput
              id="add-init"
              min={-5}
              max={40}
              value={addInit}
              onChange={setAddInit}
            />
          </div>
          <div className="flex gap-1">
            <Button
              variant={addType === 'character' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setAddType('character')}
            >
              PC
            </Button>
            <Button
              variant={addType === 'npc' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setAddType('npc')}
            >
              NPC
            </Button>
          </div>
          <Button
            onClick={submitAdd}
            disabled={!rt.isConnected || !addLabel.trim()}
          >
            + Adicionar
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function InitiativeRow({
  entry,
  onTurn,
  onDeltaHp,
  onRemove,
}: {
  entry: InitiativeEntry
  onTurn: boolean
  onDeltaHp: (delta: number) => void
  onRemove: () => void
}) {
  return (
    <div
      className={
        'flex flex-wrap items-center gap-2 rounded-md border p-2 text-sm ' +
        (onTurn ? 'border-primary/60 bg-primary/5' : '')
      }
    >
      <Badge variant="outline">{entry.initiative}</Badge>
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium">
          {entry.label}{' '}
          <Badge
            variant={entry.type === 'character' ? 'default' : 'secondary'}
            className="ml-1"
          >
            {entry.type === 'character' ? 'PC' : 'NPC'}
          </Badge>
          {onTurn && <Badge className="ml-1">Na vez</Badge>}
        </p>
        {(entry.hpCurrent !== undefined || entry.hpMax !== undefined) && (
          <p className="text-xs text-muted-foreground">
            HP {entry.hpCurrent ?? '?'}/{entry.hpMax ?? '?'}
            {entry.mpMax !== undefined && (
              <>
                {' · '}MP {entry.mpCurrent ?? '?'}/{entry.mpMax}
              </>
            )}
          </p>
        )}
      </div>
      <div className="flex gap-1">
        <Button size="sm" variant="outline" onClick={() => onDeltaHp(-5)}>
          −5
        </Button>
        <Button size="sm" variant="outline" onClick={() => onDeltaHp(-1)}>
          −1
        </Button>
        <Button size="sm" variant="outline" onClick={() => onDeltaHp(1)}>
          +1
        </Button>
        <Button size="sm" variant="outline" onClick={() => onDeltaHp(5)}>
          +5
        </Button>
        <Button size="sm" variant="ghost" onClick={onRemove}>
          ✕
        </Button>
      </div>
    </div>
  )
}

// ─── Header + status transitions + edit ──────────────────────────

function HeaderCard({
  campaignId,
  session,
}: {
  campaignId: number
  session: Session
}) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [sessionNumber, setSessionNumber] = useState(session.sessionNumber)
  const [title, setTitle] = useState(session.title ?? '')
  const [error, setError] = useState<string | null>(null)

  const invalidateAll = () => {
    qc.invalidateQueries({
      queryKey: campaignSessionQueryOptions(campaignId, session.id).queryKey,
    })
    qc.invalidateQueries({
      queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
    })
  }

  const patch = useMutation({
    mutationFn: () =>
      api.sessions.update(campaignId, session.id, {
        sessionNumber,
        title,
      }),
    onSuccess: () => {
      invalidateAll()
      setEditing(false)
      setError(null)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao salvar')
    },
  })

  const start = useMutation({
    mutationFn: () => api.sessions.start(campaignId, session.id),
    onSuccess: invalidateAll,
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-start justify-between gap-4">
        <div className="flex-1 space-y-1">
          {editing ? (
            <div className="flex flex-wrap items-end gap-2">
              <div className="w-24 space-y-1">
                <label
                  className="text-xs font-medium"
                  htmlFor="session-number"
                >
                  Nº
                </label>
                <NumberInput
                  id="session-number"
                  min={1}
                  value={sessionNumber}
                  onChange={(v) => setSessionNumber(v)}
                />
              </div>
              <div className="min-w-[220px] flex-1 space-y-1">
                <label
                  className="text-xs font-medium"
                  htmlFor="session-title"
                >
                  Título
                </label>
                <Input
                  id="session-title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Título opcional"
                />
              </div>
            </div>
          ) : (
            <>
              <CardTitle>
                Sessão {session.sessionNumber}
                {session.title ? ` — ${session.title}` : ''}
              </CardTitle>
              <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                {session.startedAt && (
                  <span>
                    Iniciada em{' '}
                    {new Date(session.startedAt).toLocaleString('pt-BR')}
                  </span>
                )}
                {session.endedAt && (
                  <span>
                    Encerrada em{' '}
                    {new Date(session.endedAt).toLocaleString('pt-BR')}
                  </span>
                )}
              </div>
            </>
          )}
        </div>
        <div className="flex flex-col items-end gap-2">
          <StatusBadge status={session.status} />
          {!editing && (
            <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
              Editar
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {editing && (
          <>
            {error && <p className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setEditing(false)
                  setSessionNumber(session.sessionNumber)
                  setTitle(session.title ?? '')
                  setError(null)
                }}
              >
                Cancelar
              </Button>
              <Button
                disabled={patch.isPending || sessionNumber < 1}
                onClick={() => patch.mutate()}
              >
                {patch.isPending ? 'Salvando…' : 'Salvar'}
              </Button>
            </div>
          </>
        )}
        {!editing && (
          <div className="flex gap-2">
            {session.status === 'planned' && (
              <Button
                onClick={() => start.mutate()}
                disabled={start.isPending}
              >
                {start.isPending ? 'Iniciando…' : 'Iniciar sessão'}
              </Button>
            )}
            {session.status === 'active' && (
              <EndSessionButton
                campaignId={campaignId}
                sessionId={session.id}
              />
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function StatusBadge({ status }: { status: SessionStatus }) {
  const variant =
    status === 'active'
      ? 'default'
      : status === 'ended'
        ? 'secondary'
        : 'outline'
  const label =
    status === 'planned'
      ? 'Planejada'
      : status === 'active'
        ? 'Ativa'
        : 'Encerrada'
  return <Badge variant={variant}>{label}</Badge>
}

// ─── End session (confirm dialog) ────────────────────────────────

function EndSessionButton({
  campaignId,
  sessionId,
}: {
  campaignId: number
  sessionId: number
}) {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const mutation = useMutation({
    mutationFn: () => api.sessions.end(campaignId, sessionId),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: campaignSessionQueryOptions(campaignId, sessionId).queryKey,
      })
      qc.invalidateQueries({
        queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
      })
      setOpen(false)
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="destructive">Encerrar sessão</Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Encerrar sessão?</DialogTitle>
          <DialogDescription>
            Sessão encerrada não pode ser reaberta. Para continuar jogando,
            crie uma nova sessão.
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
            {mutation.isPending ? 'Encerrando…' : 'Encerrar'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── Delete session ────────────────────────────────────────────

function DeleteSessionButton({
  campaignId,
  sessionId,
  sessionNumber,
}: {
  campaignId: number
  sessionId: number
  sessionNumber: number
}) {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const mutation = useMutation({
    mutationFn: () => api.sessions.delete(campaignId, sessionId),
    onSuccess: async () => {
      qc.invalidateQueries({
        queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
      })
      setOpen(false)
      await navigate({
        to: '/campaigns/$id',
        params: { id: String(campaignId) },
      })
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="destructive" size="sm">
          Excluir sessão
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Excluir sessão {sessionNumber}?</DialogTitle>
          <DialogDescription>
            Esta ação não pode ser desfeita.
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

// ─── Notes ───────────────────────────────────────────────────────

function NotesCard({
  campaignId,
  session,
}: {
  campaignId: number
  session: Session
}) {
  const qc = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [notes, setNotes] = useState(session.notes ?? '')
  const [error, setError] = useState<string | null>(null)

  const patch = useMutation({
    mutationFn: () =>
      api.sessions.update(campaignId, session.id, { notes }),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: campaignSessionQueryOptions(campaignId, session.id)
          .queryKey,
      })
      setEditing(false)
      setError(null)
    },
    onError: (e) => {
      setError(e instanceof ApiError ? e.message : 'Erro ao salvar')
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>Notas</CardTitle>
        {!editing && (
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            Editar
          </Button>
        )}
      </CardHeader>
      <CardContent className="space-y-3 text-sm">
        {editing ? (
          <>
            <Textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={10}
              placeholder="Anote acontecimentos, decisões, XP, tesouro…"
            />
            {error && <p className="text-sm text-destructive">{error}</p>}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setEditing(false)
                  setNotes(session.notes ?? '')
                  setError(null)
                }}
              >
                Cancelar
              </Button>
              <Button
                disabled={patch.isPending}
                onClick={() => patch.mutate()}
              >
                {patch.isPending ? 'Salvando…' : 'Salvar'}
              </Button>
            </div>
          </>
        ) : session.notes ? (
          <p className="whitespace-pre-line text-muted-foreground">
            {session.notes}
          </p>
        ) : (
          <p className="text-muted-foreground">Nenhuma nota ainda.</p>
        )}
      </CardContent>
    </Card>
  )
}
