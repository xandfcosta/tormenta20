import type { Session } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import { AddMonsterDrawer } from '@/features/session/add-monster-drawer'
import { MatchPeek, MatchRail } from '@/features/session/match-rail'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { NotesCard } from '@/features/session-tracker/notes-card'

/**
 * GM's match screen. The initiative tracker is the primary surface (main
 * column); session controls — status/start/end, notes, and the destructive
 * delete — sit in a side rail on wide viewports and collapse into a bottom
 * sheet on phones so the tracker owns the screen. `myCharacterIds` still
 * scopes any GM-owned combatants for the shared vitals rule, though the GM
 * can edit every combatant.
 */
export function SessionGmView({
  campaignId,
  sessionId,
  session,
  rt,
  myCharacterIds,
}: {
  campaignId: number
  sessionId: number
  session: Session
  rt: ReturnType<typeof useSessionSocket>
  myCharacterIds: Set<number>
}) {
  return (
    <div className="mx-auto grid max-w-6xl gap-4 p-3 pb-20 sm:p-4 lg:grid-cols-[minmax(0,1fr)_20rem] lg:pb-4">
      <div className="min-w-0 space-y-4">
        <InitiativeCard rt={rt} isGm myCharacterIds={myCharacterIds} />
      </div>
      <MatchRail title="Controles da sessão" peek={<MatchPeek rt={rt} />}>
        <HeaderCard campaignId={campaignId} session={session} isGm />
        <AddMonsterDrawer rt={rt} />
        <NotesCard campaignId={campaignId} session={session} />
        <div className="flex justify-end">
          <DeleteSessionButton
            campaignId={campaignId}
            sessionId={sessionId}
            sessionNumber={session.sessionNumber}
          />
        </div>
      </MatchRail>
    </div>
  )
}
