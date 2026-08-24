import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { getRouteApi } from '@tanstack/solid-router'
import { Show, createEffect, createMemo } from 'solid-js'
import {
  campaignMembersQueryOptions,
  campaignQueryOptions,
} from '@/entities/campaign/queries'
import { campaignSessionQueryOptions } from '@/entities/session/queries'
import { meQueryOptions } from '@/entities/user/queries'
import { connectionStatus } from '@/features/session-tracker/tracker-rules'
import { ConnectionChip } from '@/shared/ui/connection-chip'
import { MatchShell } from '@/pages/sessions/match-shell'
import {
  LiveSessionStatus,
  eAMinhaVez,
  playerTurnState,
} from '@/pages/sessions/live-session-status'
import { SessionGmView } from '@/pages/sessions/session-gm-view'
import { SessionPlayerView } from '@/pages/sessions/session-player-view'
import { createCharacterVitalsSync } from '@/features/session-tracker/character-vitals-sync'
import { PresenceChips } from '@/features/session-tracker/presence-chips'
import { myCharacterIdsOf } from '@/features/session-tracker/tracker-rules'
import { usePowerUses } from '@/shared/stores/power-uses-context'
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
  const powerUses = usePowerUses()
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

  const queryClient = useQueryClient()
  // A ficha mudou no servidor (o mestre aplicou uma condição, por exemplo) e
  // quem está com ela aberta precisa saber (ALE-245). Invalidar é o suficiente:
  // quem nunca buscou aquele personagem não tem a query no cache e isto vira
  // no-op, então o aviso não gera requisição para quem não olha.
  const rt = createSessionSocket(campaignId, sessionId, {
    onCharacterChanged: (characterId) => {
      void queryClient.invalidateQueries({ queryKey: ['characters', characterId] })

    },
  })
  createTurnCue(rt.state, myCharacterIds, {
    notify: (label) =>
      announce(() =>
        toast.success(`⚔️ Sua vez, ${label}!`, {
          description: 'Seu personagem está na iniciativa.',
        }),
      ),
    sfx,
  })
  createRestCue(rt, myCharacterIds, powerUses)
  // A ficha e o card do grupo seguem o combate: sem isto o mestre bate -5 e vê
  // o número antigo a 300px de distância (ALE-122).
  createCharacterVitalsSync(rt, campaignId)

  const title = () => {
    const current = settledSession()
    if (!current) return 'Sessão'
    const mesa = settledCampaign()
    return `Sessão ${current.sessionNumber}${mesa ? ` · ${mesa.name}` : ''}`
  }

  // De quem é a vez, do ponto de vista de quem está olhando. Sobe para a página
  // porque o CABEÇALHO passou a mostrá-la (ALE-201) — a mesma derivação que a
  // cena do jogador usa, importada e não recopiada.
  const vezDoJogador = createMemo(() => playerTurnState(rt, myCharacterIds()))

  return (
    <MatchShell
      campaignId={campaignId()}
      title={title()}
      live={
        <LiveSessionStatus
          round={rt.state().round}
          turn={vezDoJogador()}
        />
      }
      minhaVez={!isGm() && eAMinhaVez(vezDoJogador())}
      /* O chip de conexão vive no CABEÇALHO desde a ALE-201, e é a terceira
         casa dele: saiu do card da iniciativa na ALE-129 e da faixa do turno na
         ALE-198, sempre pela mesma razão — uma queda silenciosa no meio do
         combate não pode depender de olhar a região certa. O cabeçalho é o
         único cromo que as DUAS cenas têm sempre na tela. */
      conexao={
        <ConnectionChip
          status={connectionStatus(rt.isConnected(), rt.error())}
          dirty={rt.hasPersistenceWarning()}
        />
      }
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
                /* O nome da campanha desce como PROP e não por uma segunda
                   query lá dentro: a página já o tem assentado, e a ficha do
                   elenco precisa dele para dizer de qual snapshot se trata
                   (ALE-212). Query nova ali seria a mesma leitura duas vezes,
                   com a de baixo podendo estar pendente enquanto esta já
                   respondeu. */
                campaignName={settledCampaign()?.name ?? ''}
                /* Mesma razão do nome: a gaveta de Regras lê a campanha, e uma
                   query nova DENTRO da cena a desanexa — foi assim que o
                   `aria-hidden` do Kobalte ficou preso na ALE-211. */
                ignoredRules={settledCampaign()?.ignoredRules ?? []}
                members={settledQuery(members) ?? []}
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

/**
 * O descanso do mestre chegando à tela de quem está na sala: o aviso, e o zerar
 * dos contadores de uso por cena/dia das fichas de quem está olhando.
 *
 * O zerar é novo e conserta um buraco que ninguém via (ALE-223). Os efeitos de
 * escopo o SERVIDOR já expirava para a mesa inteira, mas "usado 1/cena" é
 * contador LOCAL, e nada do lado do jogador o tocava quando o mestre
 * descansava. O botão "Encerrar cena" da ficha mascarava isso: o jogador
 * acabava zerando por conta própria. Com ele fora da ficha, este é o único
 * caminho — e ele só alcança as fichas de QUEM ESTÁ OLHANDO, que é o certo:
 * o contador é do navegador de cada um.
 */
function createRestCue(
  rt: ReturnType<typeof createSessionSocket>,
  myCharacterIds: () => ReadonlySet<number>,
  powerUses: ReturnType<typeof usePowerUses>,
) {
  createEffect(() => {
    const scope = rt.restFlash()
    if (!scope) return
    const day = scope === 'day'
    // Fora do `announce`: o efeito colateral no estado local acontece AGORA, e
    // só o toast precisa do microtask que separa os dois ciclos reativos.
    for (const id of myCharacterIds()) {
      if (day) powerUses.resetDay(id)
      else powerUses.resetScene(id)
    }
    // O de cena não diz mais "Descanso": desde a ALE-220 este sinal também
    // chega quando o mestre ENCERRA a cena, e anunciar um descanso que ninguém
    // pediu empataria de novo as duas palavras que a ALE-210 separou. O título
    // passou a nomear o que aconteceu com as FICHAS, que é verdade nos dois
    // caminhos. O de dia continua sendo descanso, porque devolve PV e PM.
    announce(() =>
      toast.success(day ? 'Descanso de dia' : 'Efeitos de cena limpos', {
        description: day
          ? 'PV/PM recuperados e efeitos temporários limpos.'
          : 'Os usos 1/cena e as posturas também saíram das fichas.',
      }),
    )
  })
}
