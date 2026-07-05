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
      <NotesCard campaignId={campaignId} session={session.data} />
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
