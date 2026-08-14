import { Plus, Swords, Trash2 } from 'lucide-solid'
import { For, Show, createEffect, createMemo, createSignal } from 'solid-js'
import type { InitiativeEntry, RestCondition, SessionRealtime } from '@/shared/realtime/realtime'
import { buffSpells } from '@/shared/lib/spell-cache'
import { cn } from '@/shared/lib/utils'
import { Button } from '@/shared/ui/button'
import { ResourceAdjustDialog } from '@/shared/ui/resource-adjust-dialog'
import { ConnectionChip } from '@/shared/ui/connection-chip'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'
import { Select } from '@/shared/ui/select'
import { VitalBar } from '@/shared/ui/vital-bar'
import { InitiativeRollButton } from './initiative-roll'
import { type EntryPermissions, connectionStatus, entryPermissions } from './tracker-rules'

const REST_OPTIONS: { value: RestCondition; label: string }[] = [
  { value: 'ruim', label: 'Ruim (½ nível)' },
  { value: 'normal', label: 'Normal (nível)' },
  { value: 'confortavel', label: 'Confortável (2×)' },
  { value: 'luxuosa', label: 'Luxuosa (3×)' },
]

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
}) {
  const [restCondition, setRestCondition] = createSignal<RestCondition>('normal')
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
        <Show when={props.isGm}>
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

        <Show when={props.isGm}>
          <div class="flex flex-wrap items-center gap-2">
            <Button
              size="sm"
              variant="secondary"
              disabled={!props.rt.isConnected()}
              onClick={props.rt.populateParty}
            >
              Adicionar grupo
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={!props.rt.isConnected()}
              onClick={() => props.rt.rest('scene')}
            >
              Descanso de cena
            </Button>
            <div class="flex items-center gap-1">
              <Select
                aria-label="Qualidade do descanso"
                size="sm"
                class="w-[150px]"
                options={REST_OPTIONS}
                value={REST_OPTIONS.find((o) => o.value === restCondition()) ?? null}
                onChange={(option) => setRestCondition(option?.value ?? 'normal')}
              />
              <Button
                size="sm"
                variant="outline"
                disabled={!props.rt.isConnected()}
                onClick={() => props.rt.rest('day', restCondition())}
              >
                Descanso de dia
              </Button>
            </div>
          </div>
        </Show>

        <Show when={props.rt.state().initiative.length === 0}>
          <p class="text-sm text-muted-foreground">
            {props.isGm
              ? 'Sem combatentes ainda. Adicione abaixo.'
              : 'Aguardando o mestre montar a iniciativa.'}
          </p>
        </Show>

        <div class="space-y-2">
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
                />
              )
            }}
          </For>
        </div>

        <Show when={props.isGm}>
          <AddCombatantForm rt={props.rt} />
        </Show>
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
        'flex flex-col gap-2 rounded-sm border p-2.5 text-sm sm:flex-row sm:items-center sm:gap-3',
        props.selected && 'ring-1 ring-[color:var(--primary)]',
        props.onTurn
          ? 'border-[color:var(--primary)]/60 bg-[color-mix(in_oklch,var(--primary)_6%,transparent)]'
          : 'border-border/60',
      )}
    >
      <div class="flex items-center gap-2">
        <span class="rounded-sm border border-border px-1.5 font-mono text-xs tabular-nums">
          {props.entry.initiative}
        </span>
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
        <div class="flex-1 space-y-1.5 sm:min-w-[180px]">
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

      <div class="flex flex-wrap items-center justify-end gap-1">
        <Show when={props.can.applyEffect}>
          <ApplyEffectSelect onApply={props.onApplyEffect} />
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

/**
 * The GM pushes a spell buff onto a combatant. Picking fires immediately and
 * the control resets, so the same buff can be re-applied (refreshing a scene
 * buff) — buffs are never automatic, this is the explicit GM-targets-a-player
 * affordance.
 */
function ApplyEffectSelect(props: { onApply: (spellId: string) => void }) {
  // From the primed cache: a module const would evaluate before priming.
  const options = createMemo(() => buffSpells().map((s) => ({ value: s.id, label: s.name })))
  return (
    <Select
      aria-label="Aplicar efeito"
      size="sm"
      class="h-9 w-9 justify-center p-0 sm:h-8 sm:w-8"
      options={options()}
      value={null}
      placeholder="✨"
      onChange={(option) => {
        if (option) props.onApply(option.value)
      }}
    />
  )
}

/** GM-only "add combatant" row: a name, an initiative in the playable range,
 *  and PC/NPC. Resets to a fresh NPC row after each add. */
function AddCombatantForm(props: { rt: SessionRealtime }) {
  const [label, setLabel] = createSignal('')
  const [initiative, setInitiative] = createSignal(10)
  const [type, setType] = createSignal<'character' | 'npc'>('npc')

  const trimmed = () => label().trim()
  const invalid = () => trimmed().length === 0 || trimmed().length > 60

  const submit = (event: SubmitEvent) => {
    event.preventDefault()
    if (invalid() || !props.rt.isConnected()) return
    props.rt.addEntry({ label: trimmed(), initiative: initiative(), type: type() })
    setLabel('')
    setInitiative(10)
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
