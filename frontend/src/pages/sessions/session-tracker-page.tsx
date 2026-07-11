import { getRouteApi } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Button } from '@/shared/ui/button'
import { PageChrome } from '@/shared/ui/page-chrome'
import { Skeleton } from '@/shared/ui/skeleton'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { campaignQueryOptions } from '@/entities/campaign/queries'
import { charactersQueryOptions } from '@/entities/character/queries'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { NotesCard } from '@/features/session-tracker/notes-card'

const routeApi = getRouteApi('/campaigns/$id/sessions/$sid')

export function SessionDetailPage() {
  const { id, sid } = routeApi.useParams()
  const campaignId = Number(id)
  const sessionId = Number(sid)
  const session = useQuery(
    campaignSessionQueryOptions(campaignId, sessionId),
  )
  // Role drives which controls render. Until the campaign payload
  // loads, isGm stays false so GM-only controls never flash for a
  // player. `myCharacterIds` scopes in-session vitals editing to the
  // player's own combatants (the server enforces the same rule).
  const campaign = useQuery(campaignQueryOptions(campaignId))
  const characters = useQuery(charactersQueryOptions)
  const isGm = campaign.data?.role === 'gm'
  const myCharacterIds = useMemo(
    () => new Set((characters.data ?? []).map((c) => c.id)),
    [characters.data],
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
        {isGm && (
          <DeleteSessionButton
            campaignId={campaignId}
            sessionId={sessionId}
            sessionNumber={session.data.sessionNumber}
          />
        )}
      </div>

      <HeaderCard
        campaignId={campaignId}
        session={session.data}
        isGm={isGm}
      />
      <InitiativeCard
        campaignId={campaignId}
        sessionId={sessionId}
        isGm={isGm}
        myCharacterIds={myCharacterIds}
      />
      {isGm && <NotesCard campaignId={campaignId} session={session.data} />}
    </PageChrome>
  )
}
