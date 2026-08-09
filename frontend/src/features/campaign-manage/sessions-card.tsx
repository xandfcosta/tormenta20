import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { cn } from '@/shared/lib/utils'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonRows } from '@/shared/ui/skeleton'
import type { Session } from '@/shared/api/api'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { NewSessionButton } from './new-session-button'

export function SessionsCard({
  campaignId,
  isGm,
}: {
  campaignId: number
  isGm: boolean
}) {
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  // Active session first, then the rest by most-recent number (a live session
  // is what the GM most likely wants to jump back into).
  const ordered = [...(sessions.data ?? [])].sort((a, b) => {
    if (a.status === 'active' !== (b.status === 'active'))
      return a.status === 'active' ? -1 : 1
    return b.sessionNumber - a.sessionNumber
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <SectionHeading as="h2">Sessões</SectionHeading>
        {isGm && <NewSessionButton campaignId={campaignId} />}
      </CardHeader>
      <CardContent className="space-y-2">
        {sessions.isLoading && <SkeletonRows count={3} />}
        {sessions.data?.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhuma sessão ainda.
          </p>
        )}
        {ordered.map((s) => (
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
  const isActive = session.status === 'active'
  return (
    <Link
      to="/campaigns/$id/sessions/$sid"
      params={{ id: campaignId, sid: session.id }}
    >
      <div
        className={cn(
          'flex items-center justify-between rounded-md border p-2 text-sm transition-colors hover:border-primary',
          isActive && 'border-primary bg-primary/5',
        )}
      >
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
