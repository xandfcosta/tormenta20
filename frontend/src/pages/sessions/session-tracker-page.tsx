import { getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useRef } from 'react'
import { toast } from 'sonner'
import { Skeleton } from '@/shared/ui/skeleton'
import { useSessionSocket } from '@/shared/realtime/realtime'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
} from '@/entities/campaign/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { MatchShell } from '@/features/session/match-shell'
import { SessionGmView } from '@/features/session/session-gm-view'
import { SessionPlayerView } from '@/features/session/session-player-view'
import { PresenceChips } from '@/features/session-tracker/presence-chips'

const routeApi = getRouteApi('/campaigns/$id/sessions/$sid')

export function SessionDetailPage() {
  const { id, sid } = routeApi.useParams()
  const campaignId = Number(id)
  const sessionId = Number(sid)
  const session = useQuery(campaignSessionQueryOptions(campaignId, sessionId))
  // Role drives which controls render. Until the campaign payload loads,
  // isGm stays false so GM-only controls never flash for a player.
  const campaign = useQuery(campaignQueryOptions(campaignId))
  const me = useQuery(meQueryOptions)
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const isGm = campaign.data?.role === 'gm'
  // `myCharacterIds` scopes in-session vitals editing + the player-sheet match to
  // the player's own combatants (server enforces the same rule). Derived from the
  // campaign roster — a member's character is the campaign snapshot (ALE-33), which
  // is NOT in the player's /characters list, so we match by character ownerId.
  const myCharacterIds = useMemo(
    () =>
      new Set(
        (members.data ?? [])
          .filter((m) => m.character?.ownerId === me.data?.id)
          .map((m) => m.characterId),
      ),
    [members.data, me.data?.id],
  )

  // One socket for the whole match — tracker, presence bar, and toasts all
  // read the same connection instead of each mounting their own.
  const rt = useSessionSocket(campaignId, sessionId)
  useTurnCue(rt, myCharacterIds)
  useRestFlash(rt.restFlash)

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
    <MatchShell
      campaignId={campaignId}
      title={title}
      bar={<PresenceChips users={rt.present} />}
    >
      {isGm ? (
        <SessionGmView
          campaignId={campaignId}
          sessionId={sessionId}
          session={session.data}
          rt={rt}
          myCharacterIds={myCharacterIds}
        />
      ) : (
        <SessionPlayerView
          campaignId={campaignId}
          session={session.data}
          rt={rt}
          myCharacterIds={myCharacterIds}
        />
      )}
    </MatchShell>
  )
}

/**
 * Fires a toast the instant the active combatant becomes one of the viewer's
 * own characters. The row highlight covers the persistent state; the toast is
 * the transient alert. Lives on the page so it runs once per match, not once
 * per card mount.
 */
function useTurnCue(
  rt: ReturnType<typeof useSessionSocket>,
  myCharacterIds: Set<number>,
) {
  const active =
    rt.state.turnIndex >= 0 ? rt.state.initiative[rt.state.turnIndex] : undefined
  const isMyTurn =
    active?.characterId !== undefined && myCharacterIds.has(active.characterId)
  const wasMyTurn = useRef(false)
  useEffect(() => {
    if (isMyTurn && !wasMyTurn.current) {
      toast(`⚔️ Sua vez, ${active?.label}!`, {
        description: 'Seu personagem está na iniciativa.',
      })
    }
    wasMyTurn.current = isMyTurn
  }, [isMyTurn, active?.label])
}

/** GM rest broadcast → toast for everyone in the room. */
function useRestFlash(restFlash: 'day' | 'scene' | null) {
  useEffect(() => {
    if (!restFlash) return
    const day = restFlash === 'day'
    toast(`Descanso de ${day ? 'dia' : 'cena'}`, {
      description: day
        ? 'PV/PM recuperados e efeitos temporários limpos.'
        : 'Efeitos temporários de cena foram limpos.',
    })
  }, [restFlash])
}
