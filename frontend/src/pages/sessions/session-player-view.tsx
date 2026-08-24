import { useQuery } from '@tanstack/solid-query'
import { Show, createMemo, createSignal } from 'solid-js'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { CharacterSheetSkeleton } from '@/features/character-sheet/character-sheet-skeleton'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { PartyRoster } from '@/features/session-tracker/party-roster'
import { connectionStatus } from '@/features/session-tracker/tracker-rules'
import { ConnectionChip } from '@/shared/ui/connection-chip'
import type { Session } from '@/shared/api/api'
import { settledQuery } from '@/shared/lib/settled-query'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import { BoardRegion } from './board-region'
import { playerTurnState } from './live-session-status'
import { type PlayerSurface, PlayerSurfaceSwitch } from './player-surface-switch'

/**
 * A cena do jogador (ALE-30, refeita na ALE-129).
 *
 * TRÊS superfícies e um seletor ancorado no topo: minha ficha, a mesa, o
 * tabuleiro. Cada uma ocupa a tela inteira, porque a premissa do app é tela
 * cheia sem ninguém rolando a página atrás de menu — e porque o rail de 22rem
 * espremia a sessão a ponto de o tabuleiro caber em 8×4 quadrados e o Grupo
 * truncar os nomes.
 *
 * Isto REVERTE a decisão da ALE-30 de a ficha ser sempre a superfície primária.
 * Ela continua sendo a que abre por padrão — mas deixa de ser a única que cabe.
 */
export function SessionPlayerView(props: {
  campaignId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const members = useQuery(() => campaignMembersQueryOptions(props.campaignId))
  const [surface, setSurface] = createSignal<PlayerSurface>('ficha')

  // A player may own many characters but joins a campaign with exactly one:
  // it is the roster member whose characterId is theirs.
  const myCharacterId = () =>
    settledQuery(members)?.find((member) => props.myCharacterIds.has(member.characterId))
      ?.characterId

  // Enquanto a lista de membros não assenta, a cena assume que HÁ ficha e mostra
  // o esqueleto dela — piscar "você não tem personagem" para quem tem seria pior
  // que esperar.
  //
  // O que NÃO se pode fazer é deixar esta decisão trocar um ramo que MONTA quem
  // consulta a mesma lista: montar refaz a consulta, a consulta volta a
  // "carregando", os ramos se alternam e a aba morre de memória. Foi assim que a
  // cena antiga se comportava com o servidor fora do ar (ALE-143). Por isso a
  // mesa não é mais o "outro lado" desta decisão: ela é uma superfície irmã.
  const hasSheet = () => members.isPending || myCharacterId() !== undefined

  const turn = createMemo(() => playerTurnState(props.rt, props.myCharacterIds))
  const isMyTurn = () => turn().kind === 'mine'

  const boardView = createBoardViewport()

  const activeEntryId = () => {
    const live = props.rt.state()
    return live.turnIndex >= 0 ? (live.initiative[live.turnIndex]?.id ?? null) : null
  }

  // A primary ring frames the play surface so the screen reads as an active
  // session, never as plain sheet editing (ALE-30); it brightens on the
  // player's turn, matching the banner.
  const frame = () =>
    cn(
      'min-h-0 rounded-none ring-1 transition-shadow',
      isMyTurn() ? 'ring-2 ring-grimorio-gold/60' : 'ring-grimorio-gold/20',
    )

  return (
    // UMA estrutura só: o seletor existe SEMPRE. Antes, quem não tinha
    // personagem na mesa caía num ramo alternativo sem seletor nenhum — ficava
    // sem tabuleiro e sem escolha, o que é pior do que não ter ficha.
    <div class="flex h-full min-h-0 flex-col">
      {/* Ancorado, nunca rolando para fora: é o menu da cena. O estado da
          conexão vem junto porque ele vale para a sessão INTEIRA — antes ele
          morava dentro do card da iniciativa, e sumia da tela junto com ele
          quando o jogador estava olhando a ficha (ALE-129). */}
      <div class="flex shrink-0 items-center gap-2 px-3 pt-2 sm:px-4">
        <PlayerSurfaceSwitch surface={surface()} onSurface={setSurface} />
        <ConnectionChip
          status={connectionStatus(props.rt.isConnected(), props.rt.error())}
          dirty={props.rt.hasPersistenceWarning()}
        />
      </div>

      <Show when={surface() === 'ficha'}>
        <Show when={hasSheet()} fallback={<SemPersonagem />}>
          <div class={cn('min-h-0 flex-1 p-2 sm:p-3', frame())}>
            <PlayerSheet characterId={myCharacterId()} />
          </div>
        </Show>
      </Show>

      <Show when={surface() === 'mesa'}>
        <SessionTable
          campaignId={props.campaignId}
          sessionId={props.session.id}
          rt={props.rt}
          myCharacterIds={props.myCharacterIds}
        />
      </Show>

      <Show when={surface() === 'tabuleiro'}>
        <div class="flex min-h-0 flex-1 flex-col p-2 sm:p-3">
          <BoardRegion
            rt={props.rt}
            isGm={false}
            view={boardView}
            activeEntryId={activeEntryId()}
            myCharacterIds={props.myCharacterIds}
          />
        </div>
      </Show>
    </div>
  )
}

/** Quem entrou na sessão sem personagem no elenco continua vendo a mesa e o
 *  tabuleiro — só não tem ficha para abrir, e a tela diz isso. */
function SemPersonagem() {
  return (
    <div class="flex min-h-0 flex-1 items-center justify-center p-6 text-center">
      <p class="text-sm text-muted-foreground">
        Você não tem personagem nesta mesa. Acompanhe a sessão pela Mesa e pelo Tabuleiro.
      </p>
    </div>
  )
}

/**
 * A mesa: quem está na cena e em que ordem. Rola por dentro, nunca a página.
 *
 * É um COMPONENTE, e não uma função que devolve JSX. A diferença não é estilo:
 * chamada dentro de uma prop reativa (o `fallback` de um `Show`), uma função
 * dessas recria a árvore inteira a cada mudança de query — e como a árvore
 * assina queries, ela realimenta a si mesma. O sintoma é o processo comendo
 * memória até morrer, que foi como isto apareceu (ALE-129).
 */
function SessionTable(props: {
  campaignId: number
  sessionId: number
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  return (
    <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3 sm:p-4">
      {/* O `HeaderCard` SAIU daqui (ALE-213). Para o jogador ele dizia três
          coisas e as três já estavam na tela: o número da sessão (no topo E na
          faixa), o crachá "AO VIVO" (na faixa) e um título que ele não edita.
          Era uma quarta fileira de cromo antes do que a Mesa existe para
          mostrar. Ele continua inteiro no menu do MESTRE, onde as linhas de
          ajuste têm o que ajustar. */}
      <InitiativeCard
        rt={props.rt}
        isGm={false}
        myCharacterIds={props.myCharacterIds}
        connectionChip={false}
      />
      <PartyRoster campaignId={props.campaignId} />
      {/* O piloto Datastar (ALE-219). É um `<a>` cru e não um link do roteador
          DE PROPÓSITO: o destino não é uma rota desta aplicação, é uma página
          servida pelo Go — e uma navegação de SPA para ela não sairia do
          bundle. O piloto vive AO LADO desta tela em vez de substituí-la, que é
          o que torna a comparação um clique e a saída uma linha. */}
      <a
        href={`/piloto/mesa/${props.campaignId}/${props.sessionId}`}
        class="self-start text-xs text-muted-foreground underline underline-offset-4 outline-none hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
      >
        Abrir esta Mesa renderizada pelo servidor (piloto)
      </a>
    </div>
  )
}

/** The player's own sheet, loaded by id and kept in session chrome. */
function PlayerSheet(props: { characterId: number | undefined }) {
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
        />
      )}
    </Show>
  )
}

