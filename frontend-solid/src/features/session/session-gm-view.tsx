import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { NotesCard } from '@/features/session-tracker/notes-card'
import { PartyRoster } from '@/features/session-tracker/party-roster'
import { AddMonsterPanel } from './add-monster-panel'
import { EncounterPanel } from './encounter-panel'
import { MatchPeek, MatchRail } from './match-rail'

/**
 * The GM's match screen. The tracker is the primary surface; session controls
 * — status, notes, the destructive delete — sit in a rail beside it on wide
 * viewports and collapse into a bottom sheet on phones, so the tracker owns
 * the screen where the screen is small.
 */
export function SessionGmView(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  return (
    <div class="mx-auto grid max-w-6xl gap-4 p-3 pb-20 sm:p-4 lg:grid-cols-[minmax(0,1fr)_20rem] lg:pb-4">
      <div class="min-w-0 space-y-4">
        <InitiativeCard rt={props.rt} isGm myCharacterIds={props.myCharacterIds} />
        <PartyRoster campaignId={props.campaignId} />
      </div>
      <MatchRail title="Controles da sessão" peek={<MatchPeek rt={props.rt} />}>
        {/* The GM's reach into the Mesa without leaving the match (ALE-75). */}
        <div class="space-y-2">
          <AddMonsterPanel rt={props.rt} />
          <EncounterPanel rt={props.rt} />
        </div>
        <HeaderCard campaignId={props.campaignId} session={props.session} isGm />
        <NotesCard campaignId={props.campaignId} session={props.session} />
        <div class="flex justify-end">
          <DeleteSessionButton
            campaignId={props.campaignId}
            sessionId={props.sessionId}
            sessionNumber={props.session.sessionNumber}
          />
        </div>
      </MatchRail>
    </div>
  )
}
