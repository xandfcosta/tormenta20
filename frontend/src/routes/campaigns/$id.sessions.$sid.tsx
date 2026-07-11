import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/ui/button'
import { PageChrome } from '@/shared/ui/page-chrome'
import { Skeleton } from '@/shared/ui/skeleton'
import {
  campaignSessionQueryOptions,
  meQueryOptions,
} from '@/shared/lib/queries'
import { DeleteSessionButton } from './session-tracker/delete-session-button'
import { HeaderCard } from './session-tracker/header-card'
import { InitiativeCard } from './session-tracker/initiative-card'
import { NotesCard } from './session-tracker/notes-card'

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

  if (session.isLoading)
    return (
      <PageChrome width="compact" className="space-y-4">
        <Skeleton className="h-8 w-52" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-40 w-full" />
      </PageChrome>
    )
  if (session.isError)
    return (
      <PageChrome width="compact">
        <p className="text-destructive">
          {(session.error as Error).message}
        </p>
      </PageChrome>
    )
  if (!session.data) return null

  return (
    <PageChrome width="compact" className="space-y-6">
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

      <HeaderCard campaignId={campaignId} session={session.data} />
      <InitiativeCard campaignId={campaignId} sessionId={sessionId} />
      <NotesCard campaignId={campaignId} session={session.data} />
    </PageChrome>
  )
}
