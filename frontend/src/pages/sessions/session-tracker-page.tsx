import { getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { Skeleton } from '@/shared/ui/skeleton'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { campaignQueryOptions } from '@/entities/campaign/queries'
import { charactersQueryOptions } from '@/entities/character/queries'
import { MatchShell } from '@/features/session/match-shell'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { NotesCard } from '@/features/session-tracker/notes-card'

const routeApi = getRouteApi('/campaigns/$id/sessions/$sid')

export function SessionDetailPage() {
  const { id, sid } = routeApi.useParams()
  const campaignId = Number(id)
  const sessionId = Number(sid)
  const session = useQuery(campaignSessionQueryOptions(campaignId, sessionId))
  // Role drives which controls render. Until the campaign payload loads,
  // isGm stays false so GM-only controls never flash for a player.
  // `myCharacterIds` scopes in-session vitals editing to the player's own
  // combatants (the server enforces the same rule).
  const campaign = useQuery(campaignQueryOptions(campaignId))
  const characters = useQuery(charactersQueryOptions)
  const isGm = campaign.data?.role === 'gm'
  const myCharacterIds = useMemo(
    () => new Set((characters.data ?? []).map((c) => c.id)),
    [characters.data],
  )

  const title = session.data
    ? `Sessão ${session.data.sessionNumber}${campaign.data ? ` · ${campaign.data.name}` : ''}`
    : 'Sessão'

  if (session.isLoading)
    return (
      <MatchShell campaignId={campaignId} title="Carregando sessão…">
        <div className="space-y-4 p-3 sm:p-4">
          <Skeleton className="h-8 w-52" />
          <Skeleton className="h-32 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      </MatchShell>
    )
  if (session.isError)
    return (
      <MatchShell campaignId={campaignId} title="Sessão">
        <p className="p-4 text-destructive">
          {(session.error as Error).message}
        </p>
      </MatchShell>
    )
  if (!session.data) return null

  return (
    <MatchShell campaignId={campaignId} title={title}>
      <div className="mx-auto max-w-3xl space-y-4 p-3 sm:p-4">
        {isGm && (
          <div className="flex justify-end">
            <DeleteSessionButton
              campaignId={campaignId}
              sessionId={sessionId}
              sessionNumber={session.data.sessionNumber}
            />
          </div>
        )}
        <HeaderCard campaignId={campaignId} session={session.data} isGm={isGm} />
        <InitiativeCard
          campaignId={campaignId}
          sessionId={sessionId}
          isGm={isGm}
          myCharacterIds={myCharacterIds}
        />
        {isGm && <NotesCard campaignId={campaignId} session={session.data} />}
      </div>
    </MatchShell>
  )
}
