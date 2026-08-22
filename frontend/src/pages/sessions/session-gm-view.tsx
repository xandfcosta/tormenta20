import { PanelLeftOpen, Settings2, Swords } from 'lucide-solid'
import { Show, createMemo, createSignal } from 'solid-js'
import type { Session } from '@/shared/api/api'
import type { SessionRealtime } from '@/shared/realtime/realtime'
import { DeleteSessionButton } from '@/features/session-tracker/delete-session-button'
import { HeaderCard } from '@/features/session-tracker/header-card'
import { InitiativeCard } from '@/features/session-tracker/initiative-card'
import { SessionNotes } from '@/features/session-tracker/session-notes'
import { CreatureBlockDialog } from '@/features/gm-tools/creature-block-dialog'
import { blankCreatureBlock } from '@/shared/api/creature-types'
import { TurnAdvance, TurnCounter } from '@/features/session-tracker/turn-controls'
import { RestControls } from '@/features/session-tracker/rest-controls'
import { connectionStatus, palcoBaixo } from '@/features/session-tracker/tracker-rules'
import { createElementSize } from '@/shared/lib/element-size'
import { createMediaQuery } from '@/shared/lib/media-query'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import { ConnectionChip } from '@/shared/ui/connection-chip'
import { SidePanel } from '@/shared/ui/side-panel'
import { createBoardViewport } from '@/features/battle-board/board-viewport'
import { AddMonsterPanel } from './add-monster-panel'
import { BoardRegion } from './board-region'
import { CatalogPanel } from './catalog-panel'
import { CombatantDialog } from './combatant-dialog'
import { EncounterPanel } from './encounter-panel'
import { type SessionTool, GmToolRail } from './gm-tool-rail'
import { InitiativeRail } from './initiative-rail'
import { MatchControls } from './match-rail'
import { SectionTitle } from '@/shared/ui/section-label'

/** Onde cabem os dois trilhos ao lado do tabuleiro. Abaixo disto eles viram uma
 *  fileira só acima do mapa — a MESMA lista, trocando por classe. */
const RAILS_FIT = '(min-width: 1024px)'

/**
 * A cena do mestre como SHELL, com o TABULEIRO no centro (ALE-198).
 *
 * A hierarquia se inverteu, e a decisão é do dono: o que fica na tela o tempo
 * todo é o mapa, porque é o que a mesa inteira está olhando. A iniciativa só
 * acontece durante uma cena e não paga largura permanente; ficha, bestiário e
 * catálogo são CONSULTA — abrem com uma pergunta na cabeça e fecham em seguida.
 *
 * Daí a geometria: faixa do turno, trilho da fila à esquerda, tabuleiro, trilho
 * das consultas à direita. Ela **não muda de forma** — nem ao escolher um
 * combatente, nem ao abrir uma cena, nem ao cruzar 1536. Antes desta issue eram
 * quatro proporções condicionais (`threeUp` × `hasBoard` × `workspaceVazio`), e
 * a última delas movia a tela 255px em cima do clique do mestre.
 *
 * A única exceção são as NOTAS, que abrem coluna e empurram o mapa: elas se
 * escrevem enquanto se narra olhando o tabuleiro, e um overlay por cima do mapa
 * não serviria a esse gesto. É estado comandado pelo mestre, nunca pelo socket.
 *
 * Overlays são exclusivos entre si — um por vez, nunca empilhados. É o que
 * separa isto dos side sheets que a ALE-122 matou, onde um painel abria POR
 * CIMA do outro em vez de trocar.
 *
 * Abaixo de 1024 os trilhos viram uma fileira acima do mapa e a fila do combate
 * some, alcançável pelo mesmo botão. Some junto o seletor Combate/Mesa: com uma
 * superfície permanente só, não há duas regiões para alternar.
 */
export function SessionGmView(props: {
  campaignId: number
  sessionId: number
  session: Session
  rt: SessionRealtime
  myCharacterIds: ReadonlySet<number>
}) {
  const [selectedId, setSelectedId] = createSignal<string | null>(null)
  /**
   * A linha da iniciativa sob o ponteiro, para a peça dela ACENDER no mapa
   * (ALE-189). Mora aqui porque é composição: a fila é do `session-tracker` e o
   * tabuleiro é do `battle-board`, e nenhuma feature conhece a outra.
   */
  const [hoveredEntryId, setHoveredEntryId] = createSignal<string | null>(null)
  // O "NPC completo" pedido na forma de adicionar, esperando o bloco (ALE-137).
  const [pendingCreature, setPendingCreature] = createSignal<
    { label: string; initiative: number; hp: number } | undefined
  >()
  // A JANELA do tabuleiro nasce aqui e não dentro da região: ela precisa
  // sobreviver a tudo o que a cena monta e desmonta em volta.
  const boardView = createBoardViewport()
  /** A consulta aberta agora — bestiário, encontros ou catálogos. */
  const [tool, setTool] = createSignal<SessionTool | null>(null)
  /** A fila inteira, na gaveta da esquerda. */
  const [queueOpen, setQueueOpen] = createSignal(false)
  /** As notas: coluna, não overlay, e por isso estado próprio. */
  const [notesOpen, setNotesOpen] = createSignal(false)
  // Derivado do estado ao vivo: os vitais mudam a cada pancada, e uma cópia
  // mostraria o número de quando o combatente foi aberto.
  const selected = createMemo(
    () => props.rt.state().initiative.find((entry) => entry.id === selectedId()) ?? null,
  )
  const railsFit = createMediaQuery(RAILS_FIT)

  /**
   * Um overlay por vez. As notas são a exceção declarada: elas convivem com
   * qualquer coisa porque não cobrem nada — abrem coluna.
   */
  const pickTool = (escolhida: SessionTool) => {
    if (escolhida === 'notas') return setNotesOpen((aberta) => !aberta)
    setQueueOpen(false)
    setTool((atual) => (atual === escolhida ? null : escolhida))
  }
  const toolIsOpen = (consulta: SessionTool) =>
    consulta === 'notas' ? notesOpen() : tool() === consulta

  const openQueue = () => {
    setTool(null)
    setQueueOpen(true)
  }

  /**
   * Da fila OU da peça para a ficha, e é o mesmo gesto pelos dois caminhos.
   *
   * A gaveta fecha ao escolher porque o gesto termina onde começou: abriu-se a
   * fila para achar alguém, achou-se. Ajustar PV, ordem ou condição a mantém
   * aberta — isso é trabalho na fila, não escolha.
   */
  const openCombatant = (entryId: string) => {
    setSelectedId(entryId)
    setQueueOpen(false)
  }

  // A altura do PALCO, não da janela, e em JS e não em CSS: a mesma resposta
  // governa o que fica na fileira de turno E o que aparece dentro do menu da
  // sessão — e o menu nasce num portal fora do palco, onde consulta de
  // contêiner não alcança (ALE-146).
  const [palco, setPalco] = createSignal<HTMLDivElement>()
  const tamanhoDoPalco = createElementSize(palco)
  const palcoEstaBaixo = () => palcoBaixo(tamanhoDoPalco().height)

  const activeEntryId = () => {
    const live = props.rt.state()
    return live.turnIndex >= 0 ? (live.initiative[live.turnIndex]?.id ?? null) : null
  }

  return (
    <div ref={setPalco} class="flex h-full min-h-0 flex-col gap-2 p-2 sm:gap-3 sm:p-3">
      <TurnBar
        campaignId={props.campaignId}
        sessionId={props.sessionId}
        session={props.session}
        rt={props.rt}
        palcoBaixo={palcoEstaBaixo()}
      />

      {/* O "NPC completo" da ALE-137: a forma da fila emite a intenção, e é AQUI
          que o editor de bloco abre — a página é o único lugar que pode compor
          `session-tracker` com `gm-tools`. Ao salvar, a linha e o bloco nascem
          juntos e já ligados. */}
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

      {/* Coluna abaixo do degrau, fileira acima dele. Os trilhos são os MESMOS
          nós nos dois casos: um `Show` por largura seriam duas árvores para
          manter em passo, que é como um trilho e o gêmeo de telefone divergem. */}
      <div class="flex min-h-0 flex-1 flex-col gap-2 lg:flex-row">
        {/* `Show` e não `hidden lg:flex`: abaixo do degrau a fila não é uma
            forma diferente do mesmo trilho, ela simplesmente NÃO EXISTE — e o
            que não existe não deveria continuar na árvore acessível esperando
            que o CSS o esconda de quem enxerga. Quem a alcança lá é o botão da
            fileira, logo abaixo. */}
        <Show when={railsFit()}>
          <InitiativeRail
            entries={props.rt.state().initiative}
            activeEntryId={activeEntryId()}
            onOpenCombatant={openCombatant}
            onExpand={openQueue}
            onHoverEntry={setHoveredEntryId}
          />
        </Show>

        <BoardRegion
          rt={props.rt}
          isGm
          view={boardView}
          activeEntryId={activeEntryId()}
          highlightEntryId={hoveredEntryId()}
          onOpenCombatant={openCombatant}
        />

        {/* As notas EMPURRAM o mapa em vez de cobri-lo: escreve-se enquanto se
            narra olhando o tabuleiro, e é o único lugar da cena onde duas
            coisas precisam ser vistas ao mesmo tempo. Abaixo do degrau não há
            espaço para duas, e elas viram gaveta como as outras consultas. */}
        <Show when={notesOpen() && railsFit()}>
          {/* Largura PROPORCIONAL e não fixa, com piso e teto: a região precisa
              passar de 672px para o modo "lado a lado" das notas existir
              (ALE-139), e uma coluna de 26rem o mataria em toda tela. A 1920 os
              40% dão ~700px e o modo duplo cabe; a 1440 dão ~509 e a faixa
              honestamente responde "Escrever". De quebra isto inverte o achado
              da ALE-139: a região das notas ENCOLHIA quando a janela crescia,
              porque a cena repartia o workspace por proporções condicionais.
              Agora janela maior é região maior, que é o que qualquer um
              espera. */}
          <section class="grimorio-frame flex w-[clamp(22rem,40%,44rem)] shrink-0 flex-col bg-grimorio-panel p-2">
            <SessionNotes campaignId={props.campaignId} session={props.session} />
          </section>
        </Show>

        <GmToolRail
          class="order-first lg:order-last"
          isOpen={toolIsOpen}
          onPick={pickTool}
          leading={
            <Show when={!railsFit()}>
              <Button size="sm" variant="outline" class="shrink-0 gap-1.5" onClick={openQueue}>
                <PanelLeftOpen aria-hidden="true" class="size-4" />
                Iniciativa · {props.rt.state().initiative.length}
              </Button>
            </Show>
          }
        />
      </div>

      {/* A fila inteira, com dano, ordem e condição. Pela ESQUERDA, que é a
          borda de onde ela é chamada. */}
      <SidePanel
        open={queueOpen()}
        onOpenChange={setQueueOpen}
        side="left"
        title="Iniciativa"
        description="A fila do combate: ordem, vitais e condições."
      >
        {/* `fillHeight`: o cartão toma a altura da gaveta e rola SÓ a lista por
            dentro. Sem isto quem rola é o corpo do painel inteiro, e "Adicionar
            grupo" e "+ Combatente" sobem junto com a lista — o mesmo defeito
            que a ALE-131 consertou quando a fila era coluna. */}
        <InitiativeCard
          rt={props.rt}
          isGm
          fillHeight
          myCharacterIds={props.myCharacterIds}
          turnControls={false}
          heading={false}
          connectionChip={false}
          onSelect={openCombatant}
          selectedId={selectedId()}
          onDetailedAdd={setPendingCreature}
          onHoverEntry={setHoveredEntryId}
        />
      </SidePanel>

      <Show when={!railsFit()}>
        <SidePanel
          open={notesOpen()}
          onOpenChange={setNotesOpen}
          title="Notas"
          description="O que aconteceu nesta sessão."
        >
          <SessionNotes campaignId={props.campaignId} session={props.session} />
        </SidePanel>
      </Show>

      <AddMonsterPanel
        rt={props.rt}
        open={tool() === 'bestiario'}
        onOpenChange={(open) => setTool(open ? 'bestiario' : null)}
      />
      <EncounterPanel
        rt={props.rt}
        open={tool() === 'encontros'}
        onOpenChange={(open) => setTool(open ? 'encontros' : null)}
      />
      <CatalogPanel
        rt={props.rt}
        open={tool() === 'catalogos'}
        onOpenChange={(open) => setTool(open ? 'catalogos' : null)}
      />

      <CombatantDialog
        entry={selected()}
        campaignId={props.campaignId}
        onClose={() => setSelectedId(null)}
        onApplyEffect={(entryId, spellId) => props.rt.applyEffect(entryId, spellId)}
        onConditions={(entryId, conditions) => props.rt.updateEntry(entryId, { conditions })}
        onLinkCreature={(entry, creature) => {
          // A linha herda a VIDA do bloco quando não tem nenhuma: sem isto a
          // tela se contradiz — o painel dizia "não tem vida registrada" com o
          // bloco declarando 30 PV logo abaixo, e a barra do rastreador não
          // teria o que rastrear (ALE-137). Quem já tem vida não é tocado: ela
          // é estado de combate, e o bloco descreve a criatura, não o dano.
          props.rt.updateEntry(entry.id, {
            creatureId: creature.id,
            ...(entry.hpMax === undefined
              ? { hpCurrent: creature.block.hp, hpMax: creature.block.hp }
              : {}),
          })
        }}
      />
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
  /** Palco curto demais para duas fileiras de cromo (ALE-146). */
  palcoBaixo: boolean
}) {
  const state = () => props.rt.state()
  const active = () => (state().turnIndex >= 0 ? state().initiative[state().turnIndex] : undefined)

  return (
    <div class="grimorio-frame flex min-w-0 shrink-0 flex-wrap items-center gap-2 bg-grimorio-panel px-3 py-2">
      <TurnCounter state={state()} />
      {/* O estado do socket sobe para a faixa (ALE-198). Ele morava no
          cabeçalho da iniciativa, que era região permanente e virou gaveta —
          uma queda silenciosa no meio do combate passaria a parecer só um
          rastreador lento, que é exatamente o que o chip existe para impedir.
          Aqui ele fica no cromo que nunca sai da tela, e o cartão da fila
          esconde o dele (`connectionChip={false}`) para não haver dois. */}
      <ConnectionChip
        status={connectionStatus(props.rt.isConnected(), props.rt.error())}
        dirty={props.rt.hasPersistenceWarning()}
      />
      <Show when={active()}>
        {(entry) => (
          <SectionTitle as="span" contexto="painel" class="text-sm flex min-w-0 items-center gap-1.5">
            <Swords aria-hidden="true" class="size-4 text-grimorio-crimson-bright" />
            <span class="truncate">Vez de {entry().label}</span>
          </SectionTitle>
        )}
      </Show>

      {/* `min-w-0` na cadeia inteira (ALE-184): o avanço trunca o nome do
          próximo combatente, e `truncate` só encolhe se TODO ancestral flex
          puder encolher — o `min-width: auto` padrão de um item flex é o
          min-content dele, que num `white-space: nowrap` é o texto inteiro. */}
      <div class="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-2">
        {/* O avanço VOLTA para a faixa nas duas larguras (ALE-198). O par morava
            ancorado no pé da coluna da iniciativa, encostado na lista que
            percorre (ALE-142/184) — e aquilo valia enquanto a coluna estava
            sempre na tela. Agora a fila é gaveta, e o botão mais clicado da
            sessão não pode viver atrás de um clique. Fica um só na tela: o
            cartão da fila esconde o dele quando `turnControls={false}`.

            Dois nós mutuamente exclusivos e não um: o `‹` custa 44px mais o
            vão, e abaixo de 1024 essa largura sai do orçamento de cromo da
            fileira mais disputada do app — a 844×390 ela já enrola em duas e
            cada fileira come 46px de um palco que dá 390 (ALE-146). Abaixo do
            degrau fica só o avanço, que é exatamente o que havia antes; o
            desfazer de turno continua sendo do desktop.
            Ele vem ANTES dos descansos para não encostar no "Reiniciar", que
            apaga o combate inteiro: a ALE-122 afastou os dois de propósito, e
            pôr o avanço no fim da fileira desfazia isso no celular. */}
        <TurnAdvance
          onlyNext
          class="lg:hidden"
          state={state()}
          connected={props.rt.isConnected()}
          onPrevious={props.rt.previousTurn}
          onNext={props.rt.nextTurn}
        />
        <TurnAdvance
          class="hidden lg:flex"
          state={state()}
          connected={props.rt.isConnected()}
          onPrevious={props.rt.previousTurn}
          onNext={props.rt.nextTurn}
        />
        {/* Ações rápidas do fim de cena, ao lado do turno: eram duas linhas
            dentro do menu da sessão, e o mestre descansa o grupo com muito mais
            frequência do que renomeia a sessão (ALE-122).

            Elas VOLTAM para o menu quando o palco é baixo, e isso não desfaz a
            ALE-122 — desfazê-la seria escondê-las onde há espaço. A 844×390
            eram elas que faziam esta fileira enrolar em duas, e a segunda
            fileira custava 46px de um palco onde o conteúdo já recebia 138
            (ALE-146). O mestre que descansa o grupo num celular deitado paga um
            toque a mais; o que perdia era a ficha inteira. */}
        <Show when={!props.palcoBaixo}>
          <RestControls rt={props.rt} />
        </Show>
        <MatchControls
          title="Sessão"
          trigger={(open) => (
            <Button size="sm" variant="ghost" aria-label="Configurações da sessão" onClick={open}>
              <Settings2 aria-hidden="true" class="size-4" />
            </Button>
          )}
        >
          <Show when={props.palcoBaixo}>
            <div class="mb-3 flex flex-wrap gap-2">
              <RestControls rt={props.rt} />
            </div>
          </Show>
          <HeaderCard
            campaignId={props.campaignId}
            session={props.session}
            isGm
            /* "Reiniciar" saiu da faixa de turno (ALE-184): ele apaga o combate
               inteiro e se usa quase nunca, e ocupava um lugar na fileira mais
               disputada da cena. Os DESCANSOS ficam onde estão — a ALE-122 os
               tirou do menu de propósito, porque o mestre descansa o grupo com
               muito mais frequência do que renomeia a sessão. */
            resetCombat={
              <ConfirmDialog
                title="Reiniciar o combate?"
                description="A iniciativa, a rodada e o turno voltam a zero, e os combatentes saem da lista."
                confirmLabel="Reiniciar"
                destructive
                onConfirm={props.rt.resetInitiative}
                trigger={(open) => (
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={!props.rt.isConnected()}
                    onClick={open}
                  >
                    Reiniciar
                  </Button>
                )}
              />
            }
            danger={
              <DeleteSessionButton
                campaignId={props.campaignId}
                sessionId={props.sessionId}
                sessionNumber={props.session.sessionNumber}
              />
            }
          />
        </MatchControls>
      </div>
    </div>
  )
}
