import { Settings2, Swords } from 'lucide-solid'
import { For, Show, createEffect, createMemo, createSignal } from 'solid-js'
import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { CreatureBlockDialog } from '@/features/gm-tools/creature-block-dialog'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import { TurnControls, TurnCounter } from '@/features/session-tracker/turn-controls'
import { RestControls } from '@/features/session-tracker/rest-controls'
import { createMediaQuery } from '@/shared/lib/media-query'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import { BoardRegion } from './board-region'
import { MatchControls } from './match-rail'
import { type WorkspaceTab, SessionWorkspace } from './session-workspace'

/**
 * A cena do mestre como SHELL: cabe numa tela, sem rolagem de página — que é a
 * premissa de "sensação de jogo" do app (ALE-122).
 *
 * Duas regiões e uma faixa. A faixa do turno fica FIXA no topo, porque o botão
 * mais clicado da sessão saía da tela ao rolar e "de quem é a vez" estava
 * desenhado como metadado de 13px no canto. À esquerda, a iniciativa — a única
 * coisa que rola. À direita, a mesa: combatente, bestiário, catálogos e notas
 * como abas irmãs, no lugar dos side sheets que se empilhavam.
 *
 * O card "Grupo" saiu: eram os mesmos PCs com os mesmos números que já estão na
 * iniciativa, ocupando a maior área da tela para não dizer nada novo.
 *
 * Abaixo de lg não cabem duas regiões, então uma por vez com um seletor
 * visível — nunca uma escondida atrás da outra.
 */
export function SessionGmView(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const [selectedId, setSelectedId] = createSignal<string | null>(null)
  // O "NPC completo" pedido na forma de adicionar, esperando o bloco (ALE-137).
  const [pendingCreature, setPendingCreature] = createSignal<
    { label: string; initiative: number; hp: number } | undefined
  >()
  // A JANELA do tabuleiro nasce aqui e não dentro da região: `Show` desmonta o
  // conteúdo inativo, e o enquadramento morreria a cada troca de região.
  const boardView = createBoardViewport()
  const [tab, setTab] = createSignal<WorkspaceTab>('combatente')
  const [region, setRegion] = createSignal<SceneRegion>('combate')
  // Derivado do estado ao vivo: os vitais mudam a cada pancada, e uma cópia
  // mostraria o número de quando o combatente foi aberto.
  const selected = createMemo(
    () => props.rt.state().initiative.find((entry) => entry.id === selectedId()) ?? null,
  )
  const sideBySide = createMediaQuery('(min-width: 1024px)')
  // A 1536 cabem as TRÊS regiões lado a lado; entre 1024 e 1536 o tabuleiro
  // divide o lugar com a mesa, porque uma grade de 20 quadrados abaixo de ~600px
  // deixa o quadrado menor que o alvo tocável.
  const threeUp = createMediaQuery('(min-width: 1536px)')

  const select = (entryId: string) => {
    setSelectedId((current) => (current === entryId ? null : entryId))
    setTab('combatente')
    setRegion('mesa')
  }

  // Abaixo de 1536 o tabuleiro não tem coluna própria: ele vira ABA da mesa, ao
  // lado de combatente/bestiário/catálogos/notas. Uma faixa separada
  // atravessando a tela inteira para trocar só a coluna da direita era um
  // controle desalinhado do próprio efeito — e antes disso ela chegou a oferecer
  // duas opções que desenhavam a MESMA tela (ALE-130).
  const boardHasOwnColumn = () => threeUp()
  const hasBoard = () => props.rt.board() !== null

  // Abaixo de 1536 o tabuleiro é ABA, e a aba ativa continuava sendo
  // "combatente" — o mestre abria o tabuleiro e o app não o mostrava em lugar
  // nenhum, com 830×745px dizendo "clique num combatente" (ALE-161).
  //
  // O gatilho é a TRANSIÇÃO de "sem tabuleiro" para "com tabuleiro", e não o
  // estado: reagir ao estado brigaria com o mestre que foi de propósito ao
  // bestiário com o tabuleiro aberto. E não troca se ele já escolheu um
  // combatente — o tabuleiro chega pelo socket e não pode roubar a tela de
  // quem já está olhando outra coisa.
  let tinhaTabuleiro = false
  createEffect(() => {
    const agora = hasBoard()
    const abriu = agora && !tinhaTabuleiro
    tinhaTabuleiro = agora
    if (abriu && !boardHasOwnColumn() && selectedId() === null) {
      setTab('tabuleiro')
    }
  })
  const showTracker = () => sideBySide() || region() === 'combate'
  const showWorkspace = () => sideBySide() || region() === 'mesa'

  const activeEntryId = () => {
    const live = props.rt.state()
    return live.turnIndex >= 0 ? (live.initiative[live.turnIndex]?.id ?? null) : null
  }

  return (
    <div class="flex h-full min-h-0 flex-col gap-2 p-2 sm:gap-3 sm:p-3">
      <TurnBar
        campaignId={props.campaignId}
        sessionId={props.sessionId}
        session={props.session}
        rt={props.rt}
      />

      {/* O seletor só existe onde NÃO cabem duas colunas: com as duas na tela,
          a iniciativa é a espinha e quem troca é a barra de abas da mesa. */}
      <Show when={!sideBySide()}>
        <RegionSwitch region={region()} onRegion={setRegion} />
      </Show>

      <div
        class={cn(
          'grid min-h-0 flex-1 gap-3',
          // Com o tabuleiro ABERTO o 4/9/5 está certo — a coluna do meio fica
          // cheia. O defeito era ele ser FIXO: sem tabuleiro, 954px de 1920
          // exibiam "Nenhum tabuleiro aberto" enquanto quatro dos nove nomes
          // truncavam na coluna de 424px ao lado (ALE-161). Sem tabuleiro, o
          // convite para abrir um cabe numa faixa estreita e o resto volta
          // para a iniciativa, que é quem trabalha.
          threeUp() &&
            (hasBoard()
              ? 'grid-cols-[minmax(0,4fr)_minmax(0,9fr)_minmax(0,5fr)]'
              : 'grid-cols-[minmax(0,7fr)_minmax(0,3fr)_minmax(0,5fr)]'),
          !threeUp() && sideBySide() && 'grid-cols-[minmax(0,5fr)_minmax(0,7fr)]',
        )}
      >
        <Show when={showTracker()}>
          {/* A coluna NÃO rola: quem rola é a lista, por dentro do card. Rolando
              a coluna, o cabeçalho e as ações ("Adicionar grupo", "+ Combatente")
              subiam junto e sumiam justo quando a lista ficava longa (ALE-131). */}
          <div class="flex min-h-0 min-w-0 flex-col">
            {/* O "NPC completo" da ALE-137: a forma da iniciativa emite a
                intenção, e é AQUI que o editor de bloco abre — a página é o
                único lugar que pode compor `session-tracker` com `gm-tools`.
                Ao salvar, a linha e o bloco nascem juntos e já ligados. */}
            <Show when={pendingCreature()}>
              {(seed) => (
                <CreatureBlockDialog
                  campaignId={props.campaignId}
                  seed={{
                    name: seed().label,
                    block: { ...blankCreatureBlock(), hp: seed().hp > 0 ? seed().hp : 10 },
                  }}
                  onSaved={(creature) => {
                    props.rt.addEntry({
                      label: seed().label,
                      initiative: seed().initiative,
                      type: 'npc',
                      creatureId: creature.id,
                      hpCurrent: creature.block.hp,
                      hpMax: creature.block.hp,
                    })
                    setPendingCreature(undefined)
                  }}
                  onDismiss={() => setPendingCreature(undefined)}
                />
              )}
            </Show>
            <InitiativeCard
              rt={props.rt}
              isGm
              myCharacterIds={props.myCharacterIds}
              turnControls={false}
              fillHeight
              onSelect={select}
              selectedId={selectedId()}
              onDetailedAdd={setPendingCreature}
            />
          </div>
        </Show>

        <Show when={boardHasOwnColumn()}>
          <BoardRegion rt={props.rt} isGm view={boardView} activeEntryId={activeEntryId()} />
        </Show>

        <Show when={showWorkspace()}>
          <SessionWorkspace
            campaignId={props.campaignId}
            session={props.session}
            rt={props.rt}
            tab={tab()}
            onTabChange={setTab}
            selected={selected()}
            onCloseCombatant={() => setSelectedId(null)}
            boardView={boardHasOwnColumn() ? undefined : boardView}
            activeEntryId={activeEntryId()}
          />
        </Show>
      </div>
    </div>
  )
}

/**
 * Rodada, de quem é a vez e o avanço do turno — FIXO, porque é o que muda a
 * cada dez segundos e era o primeiro a sumir ao rolar.
 *
 * "Reiniciar" pede confirmação e sai de perto do "Próximo turno": ele apaga o
 * combate inteiro e estava encostado no botão mais clicado da sessão, enquanto
 * "Excluir sessão" — muito menos provável — já tinha confirmação. A proteção
 * estava no lugar errado.
 */
function TurnBar(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
}) {
  const state = () => props.rt.state()
  const active = () => (state().turnIndex >= 0 ? state().initiative[state().turnIndex] : undefined)

  return (
    <div class="flex shrink-0 flex-wrap items-center gap-2 rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2">
      <TurnCounter state={state()} />
      <Show when={active()}>
        {(entry) => (
          <span class="flex min-w-0 items-center gap-1.5 font-heading text-sm uppercase tracking-wide text-grimorio-gold">
            <Swords aria-hidden="true" class="size-4 text-[color:var(--primary)]" />
            <span class="truncate">Vez de {entry().label}</span>
          </span>
        )}
      </Show>

      <div class="ml-auto flex flex-wrap items-center justify-end gap-2">
        {/* O par mudou de casa: ele mora no cabeçalho da INICIATIVA, ao lado da
            lista que percorre (ALE-142). O que fica aqui é só o avanço, e SÓ
            abaixo de 1024 — lá a cena mostra uma região por vez, e com "Mesa"
            aberta a iniciativa não está na tela: sem isto o mestre perderia o
            botão mais clicado da sessão justamente ao abrir a ficha de alguém.
            Nunca os dois ao mesmo tempo, porque um é `lg:hidden` e o outro
            `hidden lg:flex`.
            Ele vem ANTES dos descansos para não encostar no "Reiniciar", que
            apaga o combate inteiro: a ALE-122 afastou os dois de propósito, e
            pôr o avanço no fim da fileira desfazia isso no celular. */}
        <TurnControls
          onlyNext
          class="lg:hidden"
          connected={props.rt.isConnected()}
          onPrevious={props.rt.previousTurn}
          onNext={props.rt.nextTurn}
        />
        {/* Ações rápidas do fim de cena, ao lado do turno: eram duas linhas
            dentro do menu da sessão, e o mestre descansa o grupo com muito mais
            frequência do que renomeia a sessão (ALE-122). */}
        <RestControls rt={props.rt} />
        <ConfirmDialog
          title="Reiniciar o combate?"
          description="A iniciativa, a rodada e o turno voltam a zero, e os combatentes saem da lista."
          confirmLabel="Reiniciar"
          destructive
          onConfirm={props.rt.resetInitiative}
          trigger={(open) => (
            <Button size="sm" variant="outline" disabled={!props.rt.isConnected()} onClick={open}>
              Reiniciar
            </Button>
          )}
        />
        <MatchControls
          title="Sessão"
          trigger={(open) => (
            <Button size="sm" variant="ghost" aria-label="Configurações da sessão" onClick={open}>
              <Settings2 aria-hidden="true" class="size-4" />
            </Button>
          )}
        >
          <HeaderCard campaignId={props.campaignId} session={props.session} isGm />
          <div class="flex justify-end">
            <DeleteSessionButton
              campaignId={props.campaignId}
              sessionId={props.sessionId}
              sessionNumber={props.session.sessionNumber}
            />
          </div>
        </MatchControls>
      </div>
    </div>
  )
}

/** As duas regiões da cena onde só cabe uma por vez. O tabuleiro não está aqui:
 *  abaixo de 1536 ele é ABA da mesa, e não região. */
type SceneRegion = 'combate' | 'mesa'

/** Uma região por vez onde não cabem duas — visível, nunca escondida. */
function RegionSwitch(props: { region: SceneRegion; onRegion: (region: SceneRegion) => void }) {
  return (
    <div class="flex shrink-0 gap-1">
      <For each={['combate', 'mesa'] as const}>
        {(value) => (
          <Button
            size="sm"
            variant={props.region === value ? 'default' : 'outline'}
            aria-pressed={props.region === value}
            class="flex-1 capitalize"
            onClick={() => props.onRegion(value)}
          >
            {value}
          </Button>
        )}
      </For>
    </div>
  )
}
