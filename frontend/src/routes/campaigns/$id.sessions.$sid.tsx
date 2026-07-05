import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  campaignSessionQueryOptions,
  meQueryOptions,
} from '@/lib/queries'

/**
 * Session detail placeholder — full lifecycle UI (start/end, edit
 * title/notes, session-scoped state) arrives in Fase D4. Shipped here
 * so the "create session → navigate" flow from the Campaign detail
 * page lands on a real screen instead of a 404.
 */
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
  const session = useQuery(
    campaignSessionQueryOptions(Number(id), Number(sid)),
  )

  return (
    <div className="mx-auto h-full max-w-3xl space-y-6 overflow-y-auto p-6">
      <Link to="/campaigns/$id" params={{ id }}>
        <Button variant="outline" size="sm">
          ← Voltar para a campanha
        </Button>
      </Link>
      {session.isLoading && <p>Loading…</p>}
      {session.isError && (
        <p className="text-destructive">
          {(session.error as Error).message}
        </p>
      )}
      {session.data && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>
              Sessão {session.data.sessionNumber}
              {session.data.title ? ` — ${session.data.title}` : ''}
            </CardTitle>
            <Badge
              variant={
                session.data.status === 'active'
                  ? 'default'
                  : session.data.status === 'ended'
                    ? 'secondary'
                    : 'outline'
              }
            >
              {session.data.status === 'planned'
                ? 'Planejada'
                : session.data.status === 'active'
                  ? 'Ativa'
                  : 'Encerrada'}
            </Badge>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            {session.data.notes && (
              <p className="whitespace-pre-line text-muted-foreground">
                {session.data.notes}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              Início/fim, notas editáveis e status transitions virão na
              próxima fase.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
