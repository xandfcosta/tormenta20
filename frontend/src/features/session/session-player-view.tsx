import type { Session } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'

/**
 * Player's match screen. A player watches the initiative order read-only and
 * can only touch their own combatants' vitals (enforced server-side; the card
 * gates the ± buttons by `myCharacterIds`). GM-only session controls never
 * render here. The player's own character sheet becomes the main surface in a
 * follow-up phase — for now this is the shared status + read-only tracker
 * without the GM cards.
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
  return (
    <div className="mx-auto max-w-3xl space-y-4 p-3 sm:p-4">
      <HeaderCard campaignId={campaignId} session={session} isGm={false} />
      <InitiativeCard rt={rt} isGm={false} myCharacterIds={myCharacterIds} />
    </div>
  )
}
