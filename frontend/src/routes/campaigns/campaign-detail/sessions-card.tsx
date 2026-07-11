import { Link, useNavigate } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarPlus } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { api } from '@/shared/api/api'
import type { Session } from '@/shared/api/api'
import { campaignSessionsQueryOptions } from '@/shared/lib/queries'

export function SessionsCard({ campaignId }: { campaignId: number }) {
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
        <SectionHeading variant="kallyadranoch" as="h2">
          Sessões
        </SectionHeading>
        <Button
          size="sm"
          disabled={mutation.isPending}
          onClick={() => mutation.mutate()}
        >
          <CalendarPlus className="mr-1 size-4" />
          {mutation.isPending ? 'Criando…' : `Sessão ${nextNumber}`}
        </Button>
      </CardHeader>
      <CardContent className="space-y-2">
        {sessions.isLoading && <SkeletonRows count={3} />}
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
      <div className="flex items-center justify-between rounded-md border p-2 text-sm transition hover:border-[color:var(--primary)]/50">
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
