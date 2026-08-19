import { Eye, EyeOff, Pencil, Plus, Swords, Trash2, X } from 'lucide-solid'
import { For, Show, createEffect, createSignal } from 'solid-js'
import type { InitiativeEntry, SessionRealtime } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ResourceAdjustDialog } from '@/shared/ui/resource-adjust-dialog'
import { ConnectionChip } from '@/shared/ui/connection-chip'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { VitalBar } from '@/shared/ui/vital-bar'
import { InitiativeEditDialog } from './initiative-edit-dialog'
import { InitiativeRollButton } from './initiative-roll'
import { TurnControls, TurnCounter } from './turn-controls'
import { toast } from '@/shared/ui/sonner'
import { createPartyFeedback } from './party-feedback'
import {
  type ActionVerb,
  type EntryPermissions,
  connectionStatus,
  entryPermissions,
  reservedVerbs,
} from './tracker-rules'

/**
 * O passo de um clique. Shift multiplica por 5, como no HUD da ficha — combate
 * raramente cobra 1 de dano, e quatro botões fixos ocupavam a linha inteira sem
 * dar conta de um crítico de 23 (ALE-122).
 */
const STEP = 1
const SHIFT_STEP = 5

/**
 * The initiative tracker: the primary surface of a live session. Everyone sees
 * the same order and the same bars; what changes by role is what you may TOUCH
 * (see `entryPermissions`).
 */
export function InitiativeCard(props: {
  rt: SessionRealtime
  isGm: boolean
  myCharacterIds: ReadonlySet<number>
  /** Abrir um combatente no painel ao lado. Ausente = ninguém seleciona nada. */
  onSelect?: (entryId: string) => void
  selectedId?: string | null
  /** Falso quando a cena já tem a faixa de turno fixa — senão o mesmo
   *  "Próximo turno" aparece duas vezes na tela (ALE-122). */
  turnControls?: boolean
  /** Falso quando a cena já mostra o estado da conexão em outro lugar — dois
   *  chips a poucos centímetros seriam a mesma informação duas vezes. */
  connectionChip?: boolean
  /** O card ocupa a altura que o pai dá e rola SÓ a lista por dentro. Verdadeiro
   *  na cena do mestre, onde a coluna tem altura definida; falso no rail do
   *  jogador, que já rola por fora e não daria altura nenhuma (ALE-131). */
  fillHeight?: boolean
  /**
   * O mestre pediu um NPC COMPLETO (ALE-137). A forma só emite a intenção com o
   * que já foi digitado; quem abre o editor de bloco é a PÁGINA, porque o
   * diálogo mora em `gm-tools` e uma feature não importa outra.
   *
   * Ausente = a escolha simples/completo nem aparece.
   */
  onDetailedAdd?: (seed: { label: string; initiative: number; hp: number }) => void
  /**
   * A linha sob o ponteiro (ALE-189). A cena usa isso para ACENDER a peça
   * correspondente no tabuleiro: "agora é o Ogro" custava procurar o ogro entre
   * nove peças, com a mesa esperando, e essa busca é a operação mais repetida
   * do combate.
   *
   * Ausente = ninguém está ouvindo, e a linha não gasta um handler por peça.
   */
  onHoverEntry?: (entryId: string | null) => void
}) {
  const myCharacterId = () => [...props.myCharacterIds][0]
  const [addOpen, setAddOpen] = createSignal(false)
  const status = () => connectionStatus(props.rt.isConnected(), props.rt.error())
  // Nasce UMA vez: guarda o instantâneo de quem já estava na lista e o timer da
  // espera pelo broadcast.
  const anunciarGrupo = createPartyFeedback(() => props.rt.state().initiative, toast)
  const reserved = () =>
    reservedVerbs(props.rt.state().initiative, {
      isGm: props.isGm,
      myCharacterIds: props.myCharacterIds,
    })

  return (
    <section
      class={cn(
        'rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]',
        // Sem isto, quem rola é a COLUNA inteira e o cabeçalho sobe junto: numa
        // mesa de dez combatentes, adicionar o décimo primeiro exigia rolar de
        // volta ao topo para achar o botão (ALE-131).
        props.fillHeight && 'flex h-full min-h-0 flex-col',
      )}
    >
      <header class="flex shrink-0 flex-row items-start justify-between gap-3 border-b border-grimorio-iron p-3 sm:p-4">
        {/* Uma linha só: no celular deitado o cabeçalho e a faixa de turno
            comiam metade dos 390px de altura antes de a lista começar. */}
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          <h2 class="font-heading text-lg uppercase tracking-wide text-grimorio-gold">
            Iniciativa
          </h2>
          <Show when={props.connectionChip !== false}>
            <ConnectionChip status={status()} dirty={props.rt.hasPersistenceWarning()} />
          </Show>
          {/* A rodada só sai aqui quando não há faixa de turno: com ela na tela
              seriam dois "Rodada 1" a poucos pixels um do outro (ALE-122). */}
          <Show when={props.turnControls !== false}>
            <TurnCounter state={props.rt.state()} class="text-xs" />
          </Show>
        </div>
        <div class="flex flex-wrap items-center justify-end gap-2">
          {/* O turno é uma posição NA LISTA, então o controle mora ao lado dela
              (ALE-142). Só a partir de 1024: abaixo disso a cena mostra uma
              região por vez e quem guarda o avanço é a faixa fixa, que nesse
              caso é a única na tela. Este cabeçalho já é ancorado desde a
              ALE-131 (só a lista rola), então o botão não sai da tela. */}
          <Show when={props.isGm}>
            <TurnControls
              class={props.turnControls === false ? 'hidden lg:flex' : undefined}
              connected={props.rt.isConnected()}
              onPrevious={props.rt.previousTurn}
              onNext={props.rt.nextTurn}
            />
          </Show>
          <Show when={props.isGm && props.turnControls !== false}>
            <Button
              size="sm"
              variant="outline"
              disabled={!props.rt.isConnected()}
              onClick={props.rt.resetInitiative}
            >
              Reset
            </Button>
          </Show>
        </div>
      </header>

      <div
        class={cn(
          'space-y-3 p-3 sm:p-4',
          // As AÇÕES ficam ancoradas junto do cabeçalho: elas são o que o mestre
          // procura quando a lista está longa, e eram justamente o que sumia.
          props.fillHeight && 'flex min-h-0 flex-1 flex-col pb-0 sm:pb-0',
        )}
      >
        <Show when={props.rt.error()}>
          {(message) => <p class="text-sm text-destructive">Erro realtime: {message()}</p>}
        </Show>

        <Show when={!props.isGm && myCharacterId() !== undefined}>
          <InitiativeRollButton characterId={myCharacterId()} rt={props.rt} />
        </Show>

        {/* Só "Adicionar grupo" fica: os descansos são de uma vez por sessão e
            viraram duas linhas na frente do combate — foram para o menu da
            sessão, junto do resto do que se faz raramente (ALE-122). */}
        <Show when={props.isGm}>
          <div class="flex flex-wrap gap-2">
            {/* `outline` e não `secondary`: preenchido de cinza, com o foco
                visível depois do clique, ele LIA como desabilitado — e continua
                clicável, porque `populateParty` é idempotente. O aviso conta o
                que aconteceu, que é o que faltava (ALE-135). */}
            <Button
              size="sm"
              variant="outline"
              disabled={!props.rt.isConnected()}
              onClick={() => {
                anunciarGrupo()
                props.rt.populateParty()
              }}
            >
              Adicionar grupo
            </Button>
            {/* O formulário aberto custava 118px FIXOS acima da lista: no celular
                deitado a primeira linha de combatente nascia em y=427 de uma tela
                de 390, ou seja, o mestre não via combatente nenhum. Um clique no
                topo — não um menu escondido, que foi a queixa que trouxe o
                formulário para cá (ALE-122). */}
            <Button
              size="sm"
              variant={addOpen() ? 'default' : 'outline'}
              aria-expanded={addOpen()}
              onClick={() => setAddOpen(!addOpen())}
            >
              <Show when={addOpen()} fallback={<Plus aria-hidden="true" class="size-4" />}>
                <X aria-hidden="true" class="size-4" />
              </Show>
              {addOpen() ? 'Fechar' : 'Combatente'}
            </Button>
          </div>
        </Show>

        <Show when={props.rt.state().initiative.length === 0}>
          <p class="text-sm text-muted-foreground">
            {props.isGm
              ? 'Sem combatentes ainda. Use "Adicionar grupo" ou "+ Combatente".'
              : 'Aguardando o mestre montar a iniciativa.'}
          </p>
        </Show>

        {/* Fica ABERTO enquanto o mestre adiciona vários seguidos: fechar a cada
            envio transformaria três capangas em três cliques a mais. */}
        <Show when={props.isGm && addOpen()}>
          <AddCombatantForm
            rt={props.rt}
            onClose={() => setAddOpen(false)}
            onDetailedAdd={props.onDetailedAdd}
          />
        </Show>

        {/* @container: a coluna da iniciativa é 5/12 da tela no shell do mestre
            e a tela inteira para o jogador, então a mesma largura de VIEWPORT
            dá folgas diferentes. Medido a 1024px, a linha em modo horizontal
            estourava a coluna em 141px e os botões de PV ficavam fora da área
            visível (ALE-122). */}
        {/* Só a LISTA rola. É o mesmo conserto que a barra de abas da ficha já
            recebeu (ALE-122): o cartão é fixo e só o bloco de dentro rola. */}
        <div
          class={cn(
            '@container space-y-2',
            props.fillHeight && '-mr-1 min-h-0 flex-1 overflow-y-auto pr-1 pb-3 sm:pb-4',
          )}
        >
          <For each={props.rt.state().initiative}>
            {(entry, index) => {
              const onTurn = () => index() === props.rt.state().turnIndex
              const can = () =>
                entryPermissions(entry, {
                  isGm: props.isGm,
                  myCharacterIds: props.myCharacterIds,
                })
              const isMine = () =>
                entry.characterId !== undefined && props.myCharacterIds.has(entry.characterId)
              return (
                <InitiativeRow
                  onHover={
                    props.onHoverEntry &&
                    ((dentro) => props.onHoverEntry?.(dentro ? entry.id : null))
                  }
                  onSelect={props.onSelect && (() => props.onSelect?.(entry.id))}
                  selected={props.selectedId === entry.id}
                  entry={entry}
                  onTurn={onTurn()}
                  focusOnTurn={onTurn() && isMine()}
                  can={can()}
                  reserved={reserved()}
                  onDeltaHp={(delta) => props.rt.deltaVitals(entry.id, { hpDelta: delta })}
                  onApplyEffect={(spellId) => props.rt.applyEffect(entry.id, spellId)}
                  onRemove={() => props.rt.removeEntry(entry.id)}
                  onInitiative={
                    props.isGm
                      ? (initiative) => props.rt.updateEntry(entry.id, { initiative })
                      : undefined
                  }
                  onHideHp={
                    props.isGm
                      ? (hpHidden) => props.rt.updateEntry(entry.id, { hpHidden })
                      : undefined
                  }
                />
              )
            }}
          </For>
        </div>

      </div>
    </section>
  )
}

function InitiativeRow(props: {
  /** Chamado ao entrar e ao sair do ponteiro — e também do FOCO, para quem
   *  navega por teclado ter o mesmo apontar (ALE-189). */
  onHover?: (dentro: boolean) => void
  onSelect?: () => void
  selected?: boolean
  entry: InitiativeEntry
  onTurn: boolean
  /** Scrolls into view when the viewer's OWN combatant takes its turn. */
  focusOnTurn: boolean
  can: EntryPermissions
  /** Os verbos que a LISTA reserva na coluna de ações (ALE-141). */
  reserved: ActionVerb[]
  onDeltaHp: (delta: number) => void
  onApplyEffect: (spellId: string) => void
  onRemove: () => void
  /** Ausente para quem não pode reordenar — o número vira texto. */
  onInitiative?: (initiative: number) => void
  /** Ausente para quem não decide o que a mesa vê. */
  onHideHp?: (hidden: boolean) => void
}) {
  let row: HTMLDivElement | undefined
  const hasHp = () => props.entry.hpMax !== undefined && props.entry.hpCurrent !== undefined
  const hasMp = () => props.entry.mpMax !== undefined

  createEffect(() => {
    if (props.focusOnTurn) row?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })

  return (
    <div
      ref={row}
      onMouseEnter={() => props.onHover?.(true)}
      onMouseLeave={() => props.onHover?.(false)}
      onFocusIn={() => props.onHover?.(true)}
      onFocusOut={() => props.onHover?.(false)}
      data-on-turn={props.onTurn ? 'true' : 'false'}
      class={cn(
        // Uma árvore só, quebrando por ORDEM: apertado, nome e botões dividem
        // a primeira linha e as barras passam por baixo; largo, os três viram
        // uma linha. Empilhar tudo custava uma linha por combatente.
        'flex flex-wrap items-center gap-2 rounded-sm border p-2.5 text-sm @lg:flex-nowrap @lg:gap-3',
        // `ring-inset`: o anel do Tailwind é `box-shadow`, desenhado FORA da
        // caixa e sem ocupar layout, e a lista rola dentro de um contêiner que
        // recorta o que passa das bordas dele. A linha selecionada encosta no
        // topo desse contêiner (medido: folga ZERO, e ele não tem respiro em
        // cima), então o lado de cima do anel era cortado e sobrava um traço
        // atravessado (ALE-150). Por dentro, ele não depende do vizinho.
        props.selected && 'inset-ring-1 inset-ring-[color:var(--primary)]',
        props.onTurn
          ? 'border-[color:var(--primary)]/60 bg-[color-mix(in_oklch,var(--primary)_6%,transparent)]'
          : 'border-border/60',
      )}
    >
      <div class="order-1 flex min-w-0 flex-1 items-center gap-2">
        {/* O número é BOTÃO para o mestre: "Adicionar grupo" entra com 0 e o
            conserto antes era remover e adicionar de novo, perdendo PV e
            condições no caminho (ALE-122). */}
        <Show
          when={props.onInitiative}
          fallback={
            <span class="shrink-0 rounded-sm border border-border px-1.5 font-mono text-xs tabular-nums">
              {props.entry.initiative}
            </span>
          }
        >
          {(onInitiative) => (
            <InitiativeEditDialog
              label={props.entry.label}
              current={props.entry.initiative}
              onSave={onInitiative()}
              trigger={(open) => (
                // Parecia CHIP de leitura, igual ao rótulo "NPC" ao lado, e o
                // dono concluiu da tela que a iniciativa não podia ser mudada —
                // sendo que o diálogo existe desde a ALE-122. O lápis e o
                // cursor dizem que é controle; o rótulo diz o verbo (ALE-134).
                <button
                  type="button"
                  onClick={open}
                  aria-label={`Mudar a iniciativa de ${props.entry.label}`}
                  title={`Mudar a iniciativa de ${props.entry.label}`}
                  class="flex shrink-0 cursor-pointer items-center gap-1 rounded-sm border border-border bg-[var(--grimorio-panel-raised)] px-1.5 font-mono text-xs tabular-nums transition-colors hover:border-grimorio-gold hover:text-grimorio-gold"
                >
                  {props.entry.initiative}
                  <Pencil aria-hidden="true" class="size-2.5 text-muted-foreground" />
                </button>
              )}
            />
          )}
        </Show>
        <p class="flex min-w-0 flex-wrap items-center gap-1 font-medium">
          {/* O NOME é o alvo do clique, não a linha: os botões de vitais moram
              dentro dela, e um clique de linha os engoliria (ALE-122). */}
          <Show when={props.onSelect} fallback={<span class="truncate">{props.entry.label}</span>}>
            <button
              type="button"
              class="truncate underline-offset-4 hover:underline focus-visible:underline"
              aria-pressed={props.selected}
              onClick={() => props.onSelect?.()}
            >
              {props.entry.label}
            </button>
          </Show>
          <span
            class={cn(
              'rounded-sm px-1 text-[10px] uppercase tracking-widest',
              props.entry.type === 'character'
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted text-muted-foreground',
            )}
          >
            {props.entry.type === 'character' ? 'PC' : 'NPC'}
          </span>
          <Show when={props.onTurn}>
            <span class="inline-flex items-center gap-1 rounded-sm bg-primary px-1 text-[10px] uppercase tracking-widest text-primary-foreground">
              <Swords aria-hidden="true" class="size-3" /> Na vez
            </span>
          </Show>
        </p>
      </div>

      {/* PV oculto: o jogador recebe a MARCA sem os números — "sem barra" e
          "escondido" precisam ser coisas diferentes na tela, senão o segundo
          vira silêncio e o jogador supõe (ALE-122). */}
      <Show when={props.entry.hpHidden && !hasHp()}>
        <span class="order-3 rounded-sm border border-dashed border-grimorio-iron px-1.5 text-[10px] uppercase tracking-widest text-muted-foreground">
          PV oculto
        </span>
      </Show>

      <Show when={hasHp() || hasMp()}>
        <div class="order-3 w-full min-w-0 space-y-1.5 @lg:order-2 @lg:w-44 @lg:flex-none">
          <Show when={hasHp()}>
            <VitalBar
              kind="hp"
              label="PV"
              current={props.entry.hpCurrent ?? 0}
              max={props.entry.hpMax ?? 0}
            />
          </Show>
          <Show when={hasMp()}>
            <VitalBar
              kind="mp"
              label="PM"
              current={props.entry.mpCurrent ?? 0}
              max={props.entry.mpMax ?? 0}
            />
          </Show>
        </div>
      </Show>

      {/* Quem cede espaço é o NOME, não a barra: com a barra de largura fixa e
          os botões colados à direita, as três colunas caem no mesmo X em todas
          as linhas — o serrilhado que a auditoria mediu em três posições X. Do
          outro jeito (nome fixo, barra elástica) a linha com barra estourava a
          própria caixa em 26px na coluna de 578px. */}
      {/* Cada verbo tem o SEU lugar, reservado pela lista inteira: sem isto o
          conjunto mudava por linha e a fileira encolhia, então o `+` de uma
          caía onde estava o lápis de outra e o olho não formava coluna
          (ALE-141). Quem não tem o verbo deixa o espaço vazio. */}
      <div class="order-2 ml-auto flex shrink-0 items-center justify-end gap-1 @lg:order-3">
        <Show when={props.reserved.includes('vitals') && !props.can.editVitals}>
          <span aria-hidden="true" class="h-9 w-[7.75rem] sm:h-8 sm:w-[7rem]" />
        </Show>
        <Show when={props.can.editVitals}>
          {/* O MESMO arranjo da ficha: − + e o diálogo. Antes eram quatro
              botões de passo fixo, e 23 de dano custava seis cliques ou uma
              conta de cabeça. O − vem depois do +, como no HUD, para um polegar
              apressado não curar quando queria machucar. */}
          <Button
            size="sm"
            variant="outline"
            class="h-9 min-w-9 sm:h-8 sm:min-w-8"
            aria-label={`Curar ${props.entry.label}`}
            title={`Curar ${props.entry.label} — clique 1, Shift+clique 5`}
            onClick={(event: MouseEvent) => props.onDeltaHp(event.shiftKey ? SHIFT_STEP : STEP)}
          >
            +
          </Button>
          <Button
            size="sm"
            variant="outline"
            class="h-9 min-w-9 sm:h-8 sm:min-w-8"
            aria-label={`Ferir ${props.entry.label}`}
            title={`Ferir ${props.entry.label} — clique 1, Shift+clique 5`}
            onClick={(event: MouseEvent) => props.onDeltaHp(-(event.shiftKey ? SHIFT_STEP : STEP))}
          >
            −
          </Button>
          <ResourceAdjustDialog
            label={`PV de ${props.entry.label}`}
            current={props.entry.hpCurrent ?? 0}
            max={props.entry.hpMax ?? 0}
            onSetCurrent={(next) => props.onDeltaHp(next - (props.entry.hpCurrent ?? 0))}
            triggerClass="h-9 min-w-9 sm:h-8 sm:min-w-8"
          />
        </Show>
        {/* Esconder é do MESTRE e só faz sentido em linha com vida: o olho
            fechado diz que a mesa não vê o número. */}
        <Show when={props.reserved.includes('hide') && !(hasHp() && props.onHideHp)}>
          <span aria-hidden="true" class="h-9 w-9 sm:h-8 sm:w-8" />
        </Show>
        <Show when={hasHp() ? props.onHideHp : undefined}>
          {(onHideHp) => (
            <Button
              size="sm"
              variant="ghost"
              class="h-9 w-9 sm:h-8 sm:w-8"
              aria-pressed={props.entry.hpHidden === true}
              aria-label={
                props.entry.hpHidden
                  ? `Revelar PV de ${props.entry.label}`
                  : `Ocultar PV de ${props.entry.label}`
              }
              // O ícone sozinho não conta que o que se esconde é dos JOGADORES,
              // e sem `title` quem enxerga a tela não recebia nada — o nome
              // acessível resolvia só para leitor de tela (ALE-133).
              title={
                props.entry.hpHidden
                  ? `Revelar os PV de ${props.entry.label} para os jogadores`
                  : `Ocultar os PV de ${props.entry.label} dos jogadores`
              }
              onClick={() => onHideHp()(props.entry.hpHidden !== true)}
            >
              <Show when={props.entry.hpHidden} fallback={<Eye aria-hidden="true" class="size-4" />}>
                <EyeOff aria-hidden="true" class="size-4 text-grimorio-gold" />
              </Show>
            </Button>
          )}
        </Show>
        <Show when={props.reserved.includes('remove') && !props.can.remove}>
          <span aria-hidden="true" class="h-9 w-9 sm:h-8 sm:w-8" />
        </Show>
        <Show when={props.can.remove}>
          <Button
            size="sm"
            variant="ghost"
            class="h-9 w-9 sm:h-8 sm:w-8"
            aria-label={`Remover ${props.entry.label}`}
            title={`Remover ${props.entry.label} da iniciativa`}
            onClick={() => props.onRemove()}
          >
            <Trash2 aria-hidden="true" class="size-4" />
          </Button>
        </Show>
      </div>
    </div>
  )
}

/** GM-only "add combatant" row: a name, an initiative in the playable range,
 *  PV opcional e PC/NPC. Resets to a fresh NPC row after each add. */
function AddCombatantForm(props: {
  rt: SessionRealtime
  onClose: () => void
  onDetailedAdd?: (seed: { label: string; initiative: number; hp: number }) => void
}) {
  const [label, setLabel] = createSignal('')
  const [initiative, setInitiative] = createSignal(10)
  const [hp, setHp] = createSignal(0)
  const [type, setType] = createSignal<'character' | 'npc'>('npc')
  // "Simples" é o caminho rápido do meio do combate (capanga e figurante) e
  // "completo" o do vilão recorrente — os dois caminhos declarados na hora de
  // criar, como a ALE-137 pediu. Só faz sentido em NPC: personagem já tem ficha.
  const [detailed, setDetailed] = createSignal(false)
  const wantsDetail = () => detailed() && type() === 'npc' && props.onDetailedAdd !== undefined

  const trimmed = () => label().trim()
  const invalid = () => trimmed().length === 0 || trimmed().length > 60

  const submit = (event: SubmitEvent) => {
    event.preventDefault()
    if (invalid() || !props.rt.isConnected()) return
    // PV zero significa "sem vida registrada", e a linha fica sem barra: um
    // capanga anônimo não precisa de PV, e uma barra 0/0 mentiria dizendo que
    // ele está morto. Preenchido, os botões da linha passam a fazer algo — antes
    // eles existiam e não tinham em que mexer (ALE-122).
    if (wantsDetail()) {
      // A linha nasce junto com o bloco, e quem cria as duas é a página: aqui
      // só sai o que já foi digitado, para o editor abrir preenchido.
      props.onDetailedAdd?.({ label: trimmed(), initiative: initiative(), hp: hp() })
      setLabel('')
      setInitiative(10)
      setHp(0)
      return
    }
    const vitals = hp() > 0 ? { hpCurrent: hp(), hpMax: hp() } : {}
    props.rt.addEntry({ label: trimmed(), initiative: initiative(), type: type(), ...vitals })
    setLabel('')
    setInitiative(10)
    setHp(0)
    setType('npc')
  }

  const cancelar = () => {
    setLabel('')
    setInitiative(10)
    setHp(0)
    setType('npc')
    props.onClose()
  }

  return (
    <form
      class="mt-3 flex flex-wrap items-end gap-2 rounded-sm border border-dashed border-border p-3"
      onSubmit={submit}
      // Esc é o gesto que todo mundo tenta primeiro num formulário aberto por
      // engano, e o formulário não tinha saída nenhuma além do mesmo gatilho
      // que o abriu — o que não estava dito em lugar algum (ALE-136).
      onKeyDown={(event) => {
        if (event.key !== 'Escape') return
        event.preventDefault()
        cancelar()
      }}
      noValidate
    >
      <div class="min-w-[160px] flex-1 space-y-1">
        <label for="combatant-label" class="text-[10px] uppercase tracking-widest text-muted-foreground">
          Nome
        </label>
        <Input
          id="combatant-label"
          value={label()}
          onInput={(event) => setLabel(event.currentTarget.value)}
          placeholder="Goblin salteador…"
          maxLength={60}
        />
      </div>
      <div class="w-24 space-y-1">
        <label for="combatant-initiative" class="text-[10px] uppercase tracking-widest text-muted-foreground">
          Iniciativa
        </label>
        <NumberInput
          id="combatant-initiative"
          min={-5}
          max={40}
          value={initiative()}
          onChange={setInitiative}
        />
      </div>
      <div class="w-20 space-y-1">
        <label for="combatant-hp" class="text-[10px] uppercase tracking-widest text-muted-foreground">
          PV
        </label>
        <NumberInput id="combatant-hp" min={0} max={999} value={hp()} onChange={setHp} />
      </div>
      <div class="flex gap-1">
        <Button
          type="button"
          size="sm"
          variant={type() === 'character' ? 'default' : 'outline'}
          onClick={() => setType('character')}
        >
          PC
        </Button>
        <Button
          type="button"
          size="sm"
          variant={type() === 'npc' ? 'default' : 'outline'}
          onClick={() => setType('npc')}
        >
          NPC
        </Button>
      </div>
      <Show when={type() === 'npc' && props.onDetailedAdd}>
        <div class="flex gap-1">
          <Button
            type="button"
            size="sm"
            variant={detailed() ? 'outline' : 'default'}
            aria-pressed={!detailed()}
            title="Nome, iniciativa e PV — o capanga do meio do combate"
            onClick={() => setDetailed(false)}
          >
            Simples
          </Button>
          <Button
            type="button"
            size="sm"
            variant={detailed() ? 'default' : 'outline'}
            aria-pressed={detailed()}
            title="Abre o bloco de criatura: defesa, resistências, perícias, equipamento"
            onClick={() => setDetailed(true)}
          >
            Completo
          </Button>
        </div>
      </Show>

      {/* "Adicionar" MANTÉM o formulário aberto de propósito — três capangas
          seguidos não podem custar três cliques (ALE-122). Quem fecha é o
          Cancelar, que também limpa o que foi digitado. */}
      <Button type="submit" disabled={!props.rt.isConnected() || invalid()}>
        <Plus aria-hidden="true" class="mr-1 size-4" />
        {wantsDetail() ? 'Detalhar e adicionar' : 'Adicionar'}
      </Button>
      <Button type="button" variant="ghost" onClick={cancelar}>
        Cancelar
      </Button>
    </form>
  )
}
