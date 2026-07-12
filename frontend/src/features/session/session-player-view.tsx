import { useQuery } from '@tanstack/react-query'
import { Skeleton } from '@/shared/ui/skeleton'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import type { CharacterWithComputed, Session } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import { ComputedSheetCards } from '@/features/character-sheet/computed-sheet'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'

/**
 * Player's match screen. Their own character sheet is the primary surface;
 * the session rail (status + read-only initiative) sits alongside on wide
 * viewports and on top on phones so the actionable tracker leads during
 * combat. A player only touches their own combatants' vitals (enforced
 * server-side; the card gates the ± buttons by `myCharacterIds`).
 */
export function SessionPlayerView({
  campaignId,
  session,
  rt,
  myCharacterIds,
}: {
  campaignId: number
  session: Session
  rt: ReturnType<typeof useSessionSocket>
  myCharacterIds: Set<number>
}) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  // The player's character in THIS campaign is the roster member whose
  // characterId is one of the player's own — a player can own many
  // characters but joins a campaign with exactly one.
  const myCharacterId = members.data?.find((m) =>
    myCharacterIds.has(m.characterId),
  )?.characterId

  const rail = (
    <>
      <HeaderCard campaignId={campaignId} session={session} isGm={false} />
      <InitiativeCard rt={rt} isGm={false} myCharacterIds={myCharacterIds} />
    </>
  )

  // No character on the roster yet — nothing to show as the main surface, so
  // the tracker takes the full width instead of a cramped rail.
  if (!members.isLoading && myCharacterId === undefined) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-3 sm:p-4">{rail}</div>
    )
  }

  return (
    <div className="mx-auto grid max-w-6xl gap-4 p-3 sm:p-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
      <div className="order-2 min-w-0 space-y-4 lg:order-1">
        <PlayerSheet characterId={myCharacterId} />
      </div>
      <aside className="order-1 space-y-4 lg:order-2">{rail}</aside>
    </div>
  )
}

/** Loads and renders the player's own computed sheet as the main surface. */
function PlayerSheet({ characterId }: { characterId: number | undefined }) {
  const sheet = useQuery({
    ...characterSheetQueryOptions(characterId ?? 0),
    enabled: characterId !== undefined,
  })

  if (sheet.isLoading || characterId === undefined)
    return (
      <div className="space-y-4">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  if (sheet.isError)
    return (
      <p className="text-sm text-destructive">
        {(sheet.error as Error).message}
      </p>
    )
  if (!sheet.data) return null

  const { computed } = sheet.data as CharacterWithComputed
  return <ComputedSheetCards computed={computed} />
}
