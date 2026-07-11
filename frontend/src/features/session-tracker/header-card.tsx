import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { SectionHeading } from '@/shared/ui/section-heading'
import { ApiError, api } from '@/shared/api/api'
import type { Session, SessionStatus } from '@/shared/api/api'
import { campaignSessionQueryOptions, campaignSessionsQueryOptions } from '@/entities/session/queries'
export function HeaderCard({
  campaignId,
  session,
  isGm,
}: {
  campaignId: number
  session: Session
  isGm: boolean
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
      api.sessions.update(campaignId, session.id, { sessionNumber, title }),
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
        <div className="flex-1 space-y-2">
          {editing ? (
            <div className="flex flex-wrap items-end gap-2">
              <div className="w-24 space-y-1">
                <label className="text-xs font-medium" htmlFor="session-number">
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
                <label className="text-xs font-medium" htmlFor="session-title">
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
              <SectionHeading as="h1">
                Sessão {session.sessionNumber}
                {session.title ? ` — ${session.title}` : ''}
              </SectionHeading>
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
          {isGm && !editing && (
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
        {isGm && !editing && (
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
