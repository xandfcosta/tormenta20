import { useQuery } from '@tanstack/solid-query'
import { Swords } from 'lucide-solid'
import { type JSX, Show, createMemo, createSignal } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { characterQueryOptions } from '@/entities/character/queries'
import { BOTTOM_TAB, CharacterSheet } from '@/features/character-sheet/character-sheet'
import { CharacterSheetSkeleton } from '@/features/character-sheet/character-sheet-skeleton'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { PartyRoster } from '@/features/session-tracker/party-roster'
import type { Session } from '@/shared/api/api'
import { createMediaQuery } from '@/shared/lib/media-query'
import { settledQuery } from '@/shared/lib/settled-query'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import { BoardRegion } from './board-region'
import { LiveSessionBanner, type LiveTurnState } from './live-session-banner'
import { MatchControls, MatchRail } from './match-rail'

/**
 * The player's match screen. Their own editable sheet is the primary surface —
 * the same sheet as the character page, so they play from it live. On wide
 * viewports the session rail sits beside it; on a phone there is ONE bottom
 * bar, because the session control folds into the sheet's own tab bar.
 */
export function SessionPlayerView(props: {
  campaignId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const isDesktop = createMediaQuery('(min-width: 1024px)')
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))

  // A player may own many characters but joins a campaign with exactly one:
  // it is the roster member whose characterId is theirs.
  const myCharacterId = () =>
    settledQuery(members)?.find((member) => props.myCharacterIds.has(member.characterId))
      ?.characterId

  const turn = createMemo(() => playerTurnState(props.rt, props.myCharacterIds))
  const isMyTurn = () => turn().kind === 'mine'

  const boardView = createBoardViewport()

  const activeEntryId = () => {
    const live = props.rt.state()
    return live.turnIndex >= 0 ? (live.initiative[live.turnIndex]?.id ?? null) : null
  }

  const rail = () => (
    <>
      <HeaderCard campaignId={props.campaignId} session={props.session} isGm={false} />
      <InitiativeCard rt={props.rt} isGm={false} myCharacterIds={props.myCharacterIds} />
      {/* Só aparece quando existe: um cartão dizendo "o mestre não abriu um
          tabuleiro" seria ruído permanente no rail. Nesta fatia o jogador VÊ o
          tabuleiro; mover a própria peça é a fatia do movimento (ALE-124). */}
      <Show when={props.rt.board()}>
        {/* `flex h-72` e não `min-h-64`: a área do plano é `flex-1` dentro do
            cartão, e num bloco sem altura ela colapsa para ZERO — o jogador
            recebia o tabuleiro e não via grade nenhuma. Foi o e2e de dois
            clientes que apanhou isso; em jsdom tudo mede zero e o teste de
            integração passava verde por cima (ALE-124). */}
        <div class="flex h-72">
          <BoardRegion rt={props.rt} isGm={false} view={boardView} activeEntryId={activeEntryId()} />
        </div>
      </Show>
      <PartyRoster campaignId={props.campaignId} />
    </>
  )

  // A primary ring frames the play surface so the screen reads as an active
  // session, never as plain sheet editing (ALE-30); it brightens on the
  // player's turn, matching the banner.
  const frame = () =>
    cn(
      'min-h-0 rounded-sm ring-1 transition-shadow',
      isMyTurn() ? 'ring-2 ring-[color:var(--primary)]/60' : 'ring-[color:var(--primary)]/20',
    )

  return (
    <Show
      when={members.isLoading || myCharacterId() !== undefined}
      fallback={
        // No character on the roster: nothing to put in the main surface, so
        // the tracker takes the full width instead of a cramped rail.
        <div class="mx-auto max-w-3xl space-y-4 p-3 sm:p-4">{rail()}</div>
      }
    >
      <div class="flex h-full min-h-0 flex-col">
        <LiveSessionBanner
          sessionNumber={props.session.sessionNumber}
          round={props.rt.state().round}
          turn={turn()}
        />
        <Show
          when={isDesktop()}
          fallback={
            <div class={cn('min-h-0 flex-1', frame())}>
              <PlayerSheet
                characterId={myCharacterId()}
                mobileBarSlot={<SessionBarControl>{rail()}</SessionBarControl>}
              />
            </div>
          }
        >
          <div class="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_22rem] gap-4 p-4">
            <div class={cn('min-w-0', frame())}>
              <PlayerSheet characterId={myCharacterId()} />
            </div>
            <MatchRail title="Sessão">{rail()}</MatchRail>
          </div>
        </Show>
      </div>
    </Show>
  )
}

/** The player's own sheet, loaded by id and kept in session chrome. */
function PlayerSheet(props: { characterId: number | undefined; mobileBarSlot?: JSX.Element }) {
  const [tab, setTab] = createSignal('expertises')
  const character = useQuery(() => ({
    ...characterQueryOptions(props.characterId ?? 0),
    enabled: props.characterId !== undefined,
  }))

  // Never read `character.data` while pending: that suspends, and the nearest
  // boundary is the one solid-router puts around the route match — the WHOLE
  // match screen (banner, presença, a saída) was being detached while the
  // player's own sheet loaded, leaving a blank page mid-session. The fallback
  // below was written for this moment and could never paint, because the
  // suspend happened before the `Show` was evaluated (ALE-96).
  const sheet = () => settledQuery(character)

  return (
    <Show when={sheet()} fallback={<CharacterSheetSkeleton />}>
      {(data) => (
        <CharacterSheet
          character={data()}
          tab={tab()}
          onTabChange={setTab}
          inSession
          mobileBarSlot={props.mobileBarSlot}
        />
      )}
    </Show>
  )
}

/** The session control living inside the sheet's phone bar — one bar, not two. */
function SessionBarControl(props: { children: JSX.Element }) {
  return (
    <MatchControls
      title="Sessão"
      trigger={(open) => (
        <button type="button" class={BOTTOM_TAB} onClick={open}>
          <Swords aria-hidden="true" class="size-5" />
          <span class="text-[10px]">Sessão</span>
        </button>
      )}
    >
      {props.children}
    </MatchControls>
  )
}

/** The player's turn state from the live initiative (mirrors the page's cue). */
function playerTurnState(
  rt: SessionRealtime,
  myCharacterIds: ReadonlySet<number>,
): LiveTurnState {
  const state = rt.state()
  const active = state.turnIndex >= 0 ? state.initiative[state.turnIndex] : undefined
  if (!active) return { kind: 'idle' }
  if (active.characterId !== undefined && myCharacterIds.has(active.characterId)) {
    return { kind: 'mine' }
  }
  return { kind: 'other', label: active.label }
}
