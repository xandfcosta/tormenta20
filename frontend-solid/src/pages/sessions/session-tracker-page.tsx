import { useQuery } from '@tanstack/solid-query'
import { getRouteApi } from '@tanstack/solid-router'
import { Show, createEffect, createMemo } from 'solid-js'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
} from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { MatchShell } from '@/features/session/match-shell'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { PresenceChips } from '@/features/session-tracker/presence-chips'
import { myCharacterIdsOf } from '@/features/session-tracker/tracker-rules'
import { createSessionSocket } from '@/shared/realtime/realtime'
import { Skeleton } from '@/shared/ui/skeleton'
import { toast } from '@/shared/ui/sonner'

const routeApi = getRouteApi('/campaigns/$id/sessions/$sid')

/**
 * The live session as a scene. ONE socket for the whole match — tracker,
 * presence bar and toasts read the same connection instead of each opening
 * their own.
 */
export function SessionTrackerPage() {
  const params = routeApi.useParams()
  const campaignId = () => Number(params().id)
  const sessionId = () => Number(params().sid)

  const session = useQuery(() => campaignSessionQueryOptions(campaignId(), sessionId()))
  // Role drives which controls render. While the campaign payload loads `isGm`
  // stays false, so GM-only controls never flash for a player.
  const campaign = useQuery(() => campaignQueryOptions(campaignId()))
  const members = useQuery(() => campaignMembersQueryOptions(campaignId()))
  const me = useQuery(() => meQueryOptions)

  const isGm = () => campaign.data?.role === 'gm'
  const myCharacterIds = createMemo(() => myCharacterIdsOf(members.data ?? [], me.data?.id))

  const rt = createSessionSocket(campaignId, sessionId)
  createTurnCue(rt, myCharacterIds)
  createRestCue(rt)

  const title = () =>
    session.data
      ? `Sessão ${session.data.sessionNumber}${campaign.data ? ` · ${campaign.data.name}` : ''}`
      : 'Sessão'

  return (
    <MatchShell
      campaignId={campaignId()}
      title={title()}
      bar={<PresenceChips users={rt.present()} />}
    >
      <Show
        when={!session.isLoading}
        fallback={
          <div class="space-y-4 p-3 sm:p-4">
            <Skeleton class="h-8 w-52" />
            <Skeleton class="h-32 w-full" />
            <Skeleton class="h-40 w-full" />
          </div>
        }
      >
        <Show
          when={session.data}
          fallback={
            <p class="p-4 text-destructive">{(session.error as Error | null)?.message}</p>
          }
        >
          <div class="mx-auto grid max-w-6xl gap-4 p-3 sm:p-4">
            <InitiativeCard rt={rt} isGm={isGm()} myCharacterIds={myCharacterIds()} />
          </div>
        </Show>
      </Show>
    </MatchShell>
  )
}

/**
 * Toasts the moment the active combatant becomes one of the viewer's own
 * characters. The row highlight covers the persistent state; this is the
 * transient alert — and it lives on the page so it fires once per match, not
 * once per card mount.
 */
function createTurnCue(
  rt: ReturnType<typeof createSessionSocket>,
  myCharacterIds: () => ReadonlySet<number>,
) {
  let wasMyTurn = false
  createEffect(() => {
    const state = rt.state()
    const active = state.turnIndex >= 0 ? state.initiative[state.turnIndex] : undefined
    const isMyTurn =
      active?.characterId !== undefined && myCharacterIds().has(active.characterId)
    if (isMyTurn && !wasMyTurn) {
      toast.success(`⚔️ Sua vez, ${active?.label}!`, {
        description: 'Seu personagem está na iniciativa.',
      })
    }
    wasMyTurn = isMyTurn
  })
}

/** The GM's rest broadcast → a toast for everyone in the room. */
function createRestCue(rt: ReturnType<typeof createSessionSocket>) {
  createEffect(() => {
    const scope = rt.restFlash()
    if (!scope) return
    const day = scope === 'day'
    toast.success(`Descanso de ${day ? 'dia' : 'cena'}`, {
      description: day
        ? 'PV/PM recuperados e efeitos temporários limpos.'
        : 'Efeitos temporários de cena foram limpos.',
    })
  })
}
