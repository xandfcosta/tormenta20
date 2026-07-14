import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm } from '@tanstack/react-form'
import { useState } from 'react'
import { z } from 'zod'
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
import { Field, FieldError, FieldLabel } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { SectionHeading } from '@/shared/ui/section-heading'
import { ApiError, api } from '@/shared/api/api'
import type { Session, SessionStatus } from '@/shared/api/api'
import { applyServerErrors } from '@/shared/lib/form-errors'
import { campaignSessionQueryOptions, campaignSessionsQueryOptions } from '@/entities/session/queries'

// A session is numbered (whole, ≥1) and optionally titled.
const sessionEditSchema = z.object({
  sessionNumber: z
    .number()
    .int('Nº deve ser inteiro.')
    .min(1, 'Nº deve ser ≥ 1.')
    .max(9999, 'Máximo 9999.'),
  title: z.string().max(120, 'Máximo 120 caracteres.'),
})

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
  const [formError, setFormError] = useState<string | null>(null)

  const invalidateAll = () => {
    qc.invalidateQueries({
      queryKey: campaignSessionQueryOptions(campaignId, session.id).queryKey,
    })
    qc.invalidateQueries({
      queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
    })
  }

  const form = useForm({
    defaultValues: {
      sessionNumber: session.sessionNumber,
      title: session.title ?? '',
    },
    validators: { onSubmit: sessionEditSchema },
    onSubmit: async ({ value, formApi }) => {
      setFormError(null)
      try {
        await api.sessions.update(campaignId, session.id, {
          sessionNumber: value.sessionNumber,
          title: value.title,
        })
        invalidateAll()
        setEditing(false)
      } catch (e) {
        if (!applyServerErrors(formApi, e)) {
          setFormError(e instanceof ApiError ? e.message : 'Erro ao salvar')
        }
      }
    },
  })

  const cancel = () => {
    setEditing(false)
    setFormError(null)
    form.reset()
  }

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
              <form.Field
                name="sessionNumber"
                validators={{ onChange: sessionEditSchema.shape.sessionNumber }}
              >
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field data-invalid={invalid} className="w-24">
                      <FieldLabel htmlFor={f.name}>Nº</FieldLabel>
                      <NumberInput
                        id={f.name}
                        min={1}
                        value={f.state.value}
                        onChange={(v) => f.handleChange(v)}
                        onBlur={f.handleBlur}
                        aria-invalid={invalid}
                      />
                      {invalid && <FieldError errors={f.state.meta.errors} />}
                    </Field>
                  )
                }}
              </form.Field>
              <form.Field
                name="title"
                validators={{ onChange: sessionEditSchema.shape.title }}
              >
                {(f) => {
                  const invalid = f.state.meta.isTouched && !f.state.meta.isValid
                  return (
                    <Field
                      data-invalid={invalid}
                      className="min-w-[220px] flex-1"
                    >
                      <FieldLabel htmlFor={f.name}>Título</FieldLabel>
                      <Input
                        id={f.name}
                        value={f.state.value}
                        onChange={(e) => f.handleChange(e.target.value)}
                        onBlur={f.handleBlur}
                        placeholder="Título opcional"
                        aria-invalid={invalid}
                      />
                      {invalid && <FieldError errors={f.state.meta.errors} />}
                    </Field>
                  )
                }}
              </form.Field>
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
            {formError && (
              <p className="text-sm text-destructive">{formError}</p>
            )}
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={cancel}>
                Cancelar
              </Button>
              <form.Subscribe
                selector={(s) => [s.canSubmit, s.isSubmitting] as const}
                children={([canSubmit, isSubmitting]) => (
                  <Button
                    disabled={!canSubmit || isSubmitting}
                    onClick={() => form.handleSubmit()}
                  >
                    {isSubmitting ? 'Salvando…' : 'Salvar'}
                  </Button>
                )}
              />
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
