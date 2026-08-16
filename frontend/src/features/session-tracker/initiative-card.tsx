import { Plus, Swords, Trash2 } from 'lucide-solid'
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
import { type EntryPermissions, connectionStatus, entryPermissions } from './tracker-rules'

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
}) {
  const myCharacterId = () => [...props.myCharacterIds][0]
  const status = () => connectionStatus(props.rt.isConnected(), props.rt.error())

  return (
    <section class="rounded-sm border border-grimorio-iron bg-[var(--grimorio-panel)]">
      <header class="flex flex-row items-start justify-between gap-3 border-b border-grimorio-iron p-3 sm:p-4">
        <div class="space-y-2">
          <h2 class="font-heading text-lg uppercase tracking-wide text-grimorio-gold">
            Iniciativa
          </h2>
          <div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <ConnectionChip status={status()} dirty={props.rt.hasPersistenceWarning()} />
            <span class="font-mono tabular-nums">Rodada {props.rt.state().round}</span>
          </div>
        </div>
        <Show when={props.isGm && props.turnControls !== false}>
          <div class="flex flex-wrap justify-end gap-2">
            <Button size="sm" disabled={!props.rt.isConnected()} onClick={props.rt.nextTurn}>
              Próximo turno
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={!props.rt.isConnected()}
              onClick={props.rt.resetInitiative}
            >
              Reset
            </Button>
          </div>
        </Show>
      </header>

      <div class="space-y-3 p-3 sm:p-4">
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
          <Button
            size="sm"
            variant="secondary"
            disabled={!props.rt.isConnected()}
            onClick={props.rt.populateParty}
          >
            Adicionar grupo
          </Button>
        </Show>

        <Show when={props.rt.state().initiative.length === 0}>
          <p class="text-sm text-muted-foreground">
            {props.isGm
              ? 'Sem combatentes ainda. Adicione abaixo.'
              : 'Aguardando o mestre montar a iniciativa.'}
          </p>
        </Show>

        {/* No TOPO: quem adiciona combatente faz isso ANTES de operar a lista,
            e no fim ele exigia rolar a iniciativa inteira. */}
        <Show when={props.isGm}>
          <AddCombatantForm rt={props.rt} />
        </Show>

        {/* @container: a coluna da iniciativa é 5/12 da tela no shell do mestre
            e a tela inteira para o jogador, então a mesma largura de VIEWPORT
            dá folgas diferentes. Medido a 1024px, a linha em modo horizontal
            estourava a coluna em 141px e os botões de PV ficavam fora da área
            visível (ALE-122). */}
        <div class="@container space-y-2">
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
                  onSelect={props.onSelect && (() => props.onSelect?.(entry.id))}
                  selected={props.selectedId === entry.id}
                  entry={entry}
                  onTurn={onTurn()}
                  focusOnTurn={onTurn() && isMine()}
                  can={can()}
                  onDeltaHp={(delta) => props.rt.deltaVitals(entry.id, { hpDelta: delta })}
                  onApplyEffect={(spellId) => props.rt.applyEffect(entry.id, spellId)}
                  onRemove={() => props.rt.removeEntry(entry.id)}
                  onInitiative={
                    props.isGm
                      ? (initiative) => props.rt.updateEntry(entry.id, { initiative })
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
  onSelect?: () => void
  selected?: boolean
  entry: InitiativeEntry
  onTurn: boolean
  /** Scrolls into view when the viewer's OWN combatant takes its turn. */
  focusOnTurn: boolean
  can: EntryPermissions
  onDeltaHp: (delta: number) => void
  onApplyEffect: (spellId: string) => void
  onRemove: () => void
  /** Ausente para quem não pode reordenar — o número vira texto. */
  onInitiative?: (initiative: number) => void
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
      data-on-turn={props.onTurn ? 'true' : 'false'}
      class={cn(
        // Uma árvore só, quebrando por ORDEM: apertado, nome e botões dividem
        // a primeira linha e as barras passam por baixo; largo, os três viram
        // uma linha. Empilhar tudo custava uma linha por combatente.
        'flex flex-wrap items-center gap-2 rounded-sm border p-2.5 text-sm @lg:flex-nowrap @lg:gap-3',
        props.selected && 'ring-1 ring-[color:var(--primary)]',
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
                <button
                  type="button"
                  onClick={open}
                  aria-label={`Iniciativa de ${props.entry.label}`}
                  class="shrink-0 rounded-sm border border-border px-1.5 font-mono text-xs tabular-nums transition-colors hover:border-grimorio-gold hover:text-grimorio-gold"
                >
                  {props.entry.initiative}
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
      <div class="order-2 ml-auto flex shrink-0 items-center justify-end gap-1 @lg:order-3">
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
            title="Clique = 1, Shift+clique = 5"
            onClick={(event: MouseEvent) => props.onDeltaHp(event.shiftKey ? SHIFT_STEP : STEP)}
          >
            +
          </Button>
          <Button
            size="sm"
            variant="outline"
            class="h-9 min-w-9 sm:h-8 sm:min-w-8"
            aria-label={`Ferir ${props.entry.label}`}
            title="Clique = 1, Shift+clique = 5"
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
        <Show when={props.can.remove}>
          <Button
            size="sm"
            variant="ghost"
            class="h-9 w-9 sm:h-8 sm:w-8"
            aria-label={`Remover ${props.entry.label}`}
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
function AddCombatantForm(props: { rt: SessionRealtime }) {
  const [label, setLabel] = createSignal('')
  const [initiative, setInitiative] = createSignal(10)
  const [hp, setHp] = createSignal(0)
  const [type, setType] = createSignal<'character' | 'npc'>('npc')

  const trimmed = () => label().trim()
  const invalid = () => trimmed().length === 0 || trimmed().length > 60

  const submit = (event: SubmitEvent) => {
    event.preventDefault()
    if (invalid() || !props.rt.isConnected()) return
    // PV zero significa "sem vida registrada", e a linha fica sem barra: um
    // capanga anônimo não precisa de PV, e uma barra 0/0 mentiria dizendo que
    // ele está morto. Preenchido, os botões da linha passam a fazer algo — antes
    // eles existiam e não tinham em que mexer (ALE-122).
    const vitals = hp() > 0 ? { hpCurrent: hp(), hpMax: hp() } : {}
    props.rt.addEntry({ label: trimmed(), initiative: initiative(), type: type(), ...vitals })
    setLabel('')
    setInitiative(10)
    setHp(0)
    setType('npc')
  }

  return (
    <form
      class="mt-3 flex flex-wrap items-end gap-2 rounded-sm border border-dashed border-border p-3"
      onSubmit={submit}
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
      <Button type="submit" disabled={!props.rt.isConnected() || invalid()}>
        <Plus aria-hidden="true" class="mr-1 size-4" /> Adicionar
      </Button>
    </form>
  )
}
