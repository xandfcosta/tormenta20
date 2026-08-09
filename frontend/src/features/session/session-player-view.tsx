import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Swords } from 'lucide-react'
import { Skeleton } from '@/shared/ui/skeleton'
import { cn } from '@/shared/lib/utils'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { characterQueryOptions } from '@/entities/character/queries'
import type { Character, Session } from '@/shared/api/api'
import type { useSessionSocket } from '@/shared/realtime/realtime'
import {
  BOTTOM_TAB,
  CharacterSheet,
} from '@/features/character-sheet/character-sheet'
import { MatchControls, MatchRail } from '@/features/session/match-rail'
import {
  LiveSessionBanner,
  type LiveTurnState,
} from '@/features/session/live-session-banner'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'

/**
 * Player's match screen. Their own full editable character sheet is the primary
 * surface — the same sheet as the char page, so the player edits live during
 * play. On wide viewports the session rail (status + read-only initiative) sits
 * beside the sheet. On phones there is a single bottom bar: the session control
 * is merged into the sheet's own tab bar (via `mobileBarSlot`) so two bars
 * never stack. A player only touches their own combatants' vitals (enforced
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
  const isDesktop = useMediaQuery('(min-width: 1024px)')
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  // The player's character in THIS campaign is the roster member whose
  // characterId is one of the player's own — a player can own many
  // characters but joins a campaign with exactly one.
  const myCharacterId = members.data?.find((m) =>
    myCharacterIds.has(m.characterId),
  )?.characterId

  const turn = playerTurnState(rt, myCharacterIds)
  const isMyTurn = turn.kind === 'mine'

  const rail = (
    <>
      <HeaderCard campaignId={campaignId} session={session} isGm={false} />
      <InitiativeCard
        rt={rt}
        isGm={false}
        campaignId={campaignId}
        myCharacterIds={myCharacterIds}
      />
    </>
  )

  // No character on the roster yet — nothing to show as the main surface, so
  // the tracker takes the full width instead of a cramped rail.
  if (!members.isLoading && myCharacterId === undefined) {
    return (
      <div className="mx-auto max-w-3xl space-y-4 p-3 sm:p-4">{rail}</div>
    )
  }

  // Modo Jogo: the live banner sits sticky above the player's sheet on every
  // viewport, and a primary ring frames the play surface — so the screen reads
  // as an active session, never as plain sheet editing (ALE-30). The ring
  // brightens on the player's turn to match the banner.
  const frame = cn(
    'min-h-0 rounded-lg ring-1 transition-shadow',
    isMyTurn
      ? 'ring-2 ring-[color:var(--primary)]/60'
      : 'ring-[color:var(--primary)]/20',
  )

  return (
    <div className="flex h-full min-h-0 flex-col">
      <LiveSessionBanner
        sessionNumber={session.sessionNumber}
        round={rt.state.round}
        turn={turn}
      />
      {!isDesktop ? (
        // Phone: the sheet owns the full height and hosts the session control
        // in its bottom bar — no outer grid, no separate rail bar.
        <div className={cn('min-h-0 flex-1', frame)}>
          <PlayerSheet
            characterId={myCharacterId}
            mobileBarSlot={<SessionBarControl rail={rail} />}
          />
        </div>
      ) : (
        // Desktop: sheet | rail split, both full height inside the shell body.
        <div className="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_22rem] gap-4 p-4">
          <div className={cn('min-w-0', frame)}>
            <PlayerSheet characterId={myCharacterId} />
          </div>
          <MatchRail title="Sessão">{rail}</MatchRail>
        </div>
      )}
    </div>
  )
}

/** The player's turn state derived from the live initiative (mirrors useTurnCue). */
function playerTurnState(
  rt: ReturnType<typeof useSessionSocket>,
  myCharacterIds: Set<number>,
): LiveTurnState {
  const active =
    rt.state.turnIndex >= 0 ? rt.state.initiative[rt.state.turnIndex] : undefined
  if (!active) return { kind: 'idle' }
  if (active.characterId !== undefined && myCharacterIds.has(active.characterId))
    return { kind: 'mine' }
  return { kind: 'other', label: active.label }
}

/** Session control shaped like a sheet tab, for the merged phone bottom bar. */
function SessionBarControl({ rail }: { rail: ReactNode }) {
  return (
    <MatchControls
      title="Sessão"
      trigger={
        <button type="button" aria-label="Sessão" title="Sessão" className={BOTTOM_TAB}>
          <Swords className="size-5 text-[color:var(--primary)]" />
        </button>
      }
    >
      {rail}
    </MatchControls>
  )
}

/** Loads and renders the player's own editable sheet as the main surface. */
function PlayerSheet({
  characterId,
  mobileBarSlot,
}: {
  characterId: number | undefined
  mobileBarSlot?: ReactNode
}) {
  const character = useQuery({
    ...characterQueryOptions(characterId ?? 0),
    enabled: characterId !== undefined,
  })

  if (character.isLoading || characterId === undefined)
    return (
      <div className="space-y-4 p-3">
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    )
  if (character.isError)
    return (
      <p className="p-3 text-sm text-destructive">
        {(character.error as Error).message}
      </p>
    )
  if (!character.data) return null

  return (
    <CharacterSheet
      character={character.data as Character}
      mobileBarSlot={mobileBarSlot}
      inSession
    />
  )
}
