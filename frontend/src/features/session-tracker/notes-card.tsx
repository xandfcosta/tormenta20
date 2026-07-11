import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Textarea } from '@/shared/ui/textarea'
import { ApiError, api } from '@/shared/api/api'
import type { Session } from '@/shared/api/api'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
export function NotesCard({
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
        queryKey: campaignSessionQueryOptions(campaignId, session.id).queryKey,
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
        <SectionHeading variant="aharadak" as="h2">
          Notas
        </SectionHeading>
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
