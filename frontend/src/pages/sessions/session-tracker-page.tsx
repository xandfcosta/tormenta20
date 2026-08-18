import { useQuery } from '@tanstack/solid-query'
import { getRouteApi } from '@tanstack/solid-router'
import { Show, createEffect, createMemo } from 'solid-js'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
} from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { MatchShell } from '@/pages/sessions/match-shell'
import { SessionGmView } from '@/pages/sessions/session-gm-view'
import { SessionPlayerView } from '@/pages/sessions/session-player-view'
import { createCharacterVitalsSync } from '@/features/session-tracker/character-vitals-sync'
import { PresenceChips } from '@/features/session-tracker/presence-chips'
import { myCharacterIdsOf } from '@/features/session-tracker/tracker-rules'
import { createTurnCue } from '@/features/session-tracker/turn-cue'
import { createSessionSocket } from '@/shared/realtime/realtime'
import { createSfx, createSfxToggle } from '@/shared/lib/sfx'
import { useUi } from '@/shared/stores/ui-context'
import { settledQuery } from '@/shared/lib/settled-query'
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
  const ui = useUi()
  const sfx = createSfx(ui)
  const toggleSfx = createSfxToggle(ui, sfx)

  // TODAS as leituras passam por `settledQuery` (ALE-96 de novo, um andar
  // acima): tocar `.data` de uma query PENDENTE suspende, e o `Suspense` que o
  // solid-router põe em todo route match desanexa a cena inteira — ao clicar
  // "Continuar sessão" o jogador via a tela EM BRANCO até os dados chegarem.
  // O título é o pior deles, porque é prop do MatchShell: ele é avaliado ANTES
  // do `Show`, então o Skeleton escrito para esse instante nunca podia pintar.
  const settledSession = () => settledQuery(session)
  const settledCampaign = () => settledQuery(campaign)

  const isGm = () => settledCampaign()?.role === 'gm'
  const myCharacterIds = createMemo(() =>
    myCharacterIdsOf(settledQuery(members) ?? [], settledQuery(me)?.id),
  )

  // A cena só escolhe a view quando a CAMPANHA assentou: sem isso, `isGm` seria
  // falso enquanto ela voa e o mestre veria a view do jogador piscar antes de
  // trocar — que era justamente o que o suspend escondia.
  const scene = createMemo(() => {
    const current = settledSession()
    return current && settledCampaign() ? current : null
  })

  const rt = createSessionSocket(campaignId, sessionId)
  createTurnCue(rt.state, myCharacterIds, {
    notify: (label) =>
      announce(() =>
        toast.success(`⚔️ Sua vez, ${label}!`, {
          description: 'Seu personagem está na iniciativa.',
        }),
      ),
    sfx,
  })
  createRestCue(rt)
  // A ficha e o card do grupo seguem o combate: sem isto o mestre bate -5 e vê
  // o número antigo a 300px de distância (ALE-122).
  createCharacterVitalsSync(rt, campaignId)

  const title = () => {
    const current = settledSession()
    if (!current) return 'Sessão'
    const mesa = settledCampaign()
    return `Sessão ${current.sessionNumber}${mesa ? ` · ${mesa.name}` : ''}`
  }

  return (
    <MatchShell
      campaignId={campaignId()}
      title={title()}
      bar={<PresenceChips users={rt.present()} />}
      sfxEnabled={ui.sfx()}
      onToggleSfx={toggleSfx}
    >
      <Show
        when={!session.isError}
        fallback={
          <p class="p-4 text-destructive">{(session.error as Error | null)?.message}</p>
        }
      >
        <Show when={scene()} fallback={<SessionSkeleton />}>
          {(data) => (
            <Show
              when={isGm()}
              fallback={
                <SessionPlayerView
                  campaignId={campaignId()}
                  session={data()}
                  rt={rt}
                  myCharacterIds={myCharacterIds()}
                />
              }
            >
              <SessionGmView
                campaignId={campaignId()}
                sessionId={sessionId()}
                session={data()}
                rt={rt}
                myCharacterIds={myCharacterIds()}
              />
            </Show>
          )}
        </Show>
      </Show>
    </MatchShell>
  )
}

/**
 * O lugar do conteúdo enquanto a sessão carrega. Anunciado (`role="status"`)
 * porque é o único sinal de que algo está acontecendo — e é o que prova, no
 * e2e, que a cena não ficou em branco.
 */
function SessionSkeleton() {
  return (
    <div class="space-y-4 p-3 sm:p-4" role="status" aria-label="Carregando a sessão">
      <Skeleton class="h-8 w-52" />
      <Skeleton class="h-32 w-full" />
      <Skeleton class="h-40 w-full" />
    </div>
  )
}

/**
 * Dispara o toast FORA do ciclo de atualização que o originou.
 *
 * O toast vive numa árvore reativa de terceiro (solid-sonner) que, ao montar,
 * mede a própria altura e escreve essa altura de volta num sinal. Chamado de
 * dentro de um `createEffect`, esse write entra no MESMO ciclo do efeito que o
 * chamou, e o ciclo se realimenta: medi a aba do mestre travada e o toast foi
 * medido 400 vezes com a mesma altura (73,3px), com `runUpdates` aninhado 78
 * níveis e subindo — a aba morre e a sessão vai junto (ALE-122).
 *
 * Um microtask separa os dois ciclos: o efeito termina, e só então o toast
 * monta com a árvore dele em paz.
 */
function announce(fire: () => void): void {
  queueMicrotask(fire)
}

/** The GM's rest broadcast → a toast for everyone in the room. */
function createRestCue(rt: ReturnType<typeof createSessionSocket>) {
  createEffect(() => {
    const scope = rt.restFlash()
    if (!scope) return
    const day = scope === 'day'
    announce(() =>
      toast.success(`Descanso de ${day ? 'dia' : 'cena'}`, {
        description: day
          ? 'PV/PM recuperados e efeitos temporários limpos.'
          : 'Efeitos temporários de cena foram limpos.',
      }),
    )
  })
}
