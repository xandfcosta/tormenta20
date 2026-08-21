import type {
  ChaseEventRow,
  DungeonIdea,
  RewardCastigoRow,
  RuinaRow,
} from '@/shared/api/api'
import {
  castigoLabel,
  chaseEventFromRoll,
  classifyDungeonSize,
  dungeonIdeaFromRoll,
  dungeonSizeRow,
  plannedThreats,
  rewardCastigoFromRoll,
  rewardLabel,
  ruinaFromRoll,
} from '@/features/gm-tools/improviso-rules'
import { Dices } from 'lucide-solid'
import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import { createRollHistory } from '@/features/gm-tools/roll-history'
import { rollDice } from '@/shared/lib/dice'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { NumberInput } from '@/shared/ui/number-input'

const SIZE_LABEL: Record<string, string> = {
  pequena: 'Pequena',
  media: 'Média',
  grande: 'Grande',
}

const PACING_LABEL: Record<string, string> = {
  'parte-de-sessao': 'Parte de uma sessão',
  'sessao-inteira': 'Sessão inteira',
  'aventura-inteira': 'Aventura inteira',
}

/**
 * Improviso — the Cap 6 tables and the dungeon skeleton in one tool. They were
 * two screens in the React app, but the d20 dungeon idea is a Cap 6 table that
 * appeared in BOTH, and one home for it is one place to look.
 *
 * Everything rolls client-side: these catalogs are pure data, so a GM caught
 * without an answer mid-scene gets one without a round trip.
 */
export function ImprovisoTool() {
  return (
    <section class="flex min-h-0 flex-1 flex-col gap-4" aria-labelledby="mesa-improviso">
      <h2
        id="mesa-improviso"
        class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
      >
        Improviso
      </h2>

      <div class="min-h-0 flex-1 space-y-4 overflow-y-auto pr-1">
        <div>
          <p class="mb-2 font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
            Tabelas do Cap 6
          </p>
          <div class="grid items-start gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <RuinaTable />
            <ChaseTable />
            <RewardTable />
          </div>
        </div>

        <div class="border-t border-grimorio-iron pt-4">
          <p class="mb-2 font-heading text-[11px] uppercase tracking-[0.16em] text-muted-foreground">
            Masmorra
          </p>
          <DungeonPlanner />
        </div>
      </div>
    </section>
  )
}

function RuinaTable() {
  const history = createRollHistory<RuinaRow>()
  return (
    <TableCard
      title="Ermos — Ruína"
      dice="d6"
      page="p269"
      onRoll={() => {
        const roll = rollDice(1, 6)
        history.push(roll, ruinaFromRoll(roll))
      }}
      onClear={history.clear}
      entries={history.entries()}
      render={(row) => row.label}
    />
  )
}

function ChaseTable() {
  const history = createRollHistory<ChaseEventRow>()
  return (
    <TableCard
      title="Perseguição — evento"
      dice="d20"
      page="p272"
      onRoll={() => {
        const roll = rollDice(1, 20)
        history.push(roll, chaseEventFromRoll(roll))
      }}
      onClear={history.clear}
      entries={history.entries()}
      render={(row) => (row.test ? `${row.kind} · teste: ${row.test}` : row.kind)}
    />
  )
}

function RewardTable() {
  const history = createRollHistory<RewardCastigoRow>()
  return (
    <TableCard
      title="Consequências"
      dice="d6"
      page="p276"
      onRoll={() => {
        const roll = rollDice(1, 6)
        history.push(roll, rewardCastigoFromRoll(roll))
      }}
      onClear={history.clear}
      entries={history.entries()}
      render={(row) =>
        `Recompensa: ${rewardLabel(row.reward)} · Castigo: ${castigoLabel(row.castigo)}`
      }
    />
  )
}

/** One rollable table: the die, the last draw, and the few before it. */
function TableCard<T>(props: {
  title: string
  dice: string
  page: string
  onRoll: () => void
  onClear: () => void
  entries: { roll: number; result: T }[]
  render: (result: T) => JSX.Element
}) {
  return (
    <div class="space-y-2 rounded-sm border border-grimorio-iron p-3">
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <p class="font-heading text-[11px] uppercase tracking-[0.14em] text-grimorio-gold">
          {props.title}
        </p>
        <p class="font-mono text-[10px] text-muted-foreground">
          {props.dice} · {props.page}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Button type="button" size="sm" variant="outline" onClick={props.onRoll}>
          <Dices aria-hidden="true" class="mr-1 size-4" />
          Rolar {props.dice}
        </Button>
        <Show when={props.entries.length > 0}>
          <Button type="button" size="sm" variant="ghost" onClick={props.onClear}>
            Limpar
          </Button>
        </Show>
      </div>

      <Show
        when={props.entries.length > 0}
        fallback={<p class="text-[11px] text-muted-foreground">Ainda não rolado.</p>}
      >
        <ul class="space-y-1">
          <For each={props.entries}>
            {(entry, index) => (
              <li
                class="flex gap-2 text-[11px]"
                classList={{ 'text-foreground': index() === 0, 'text-muted-foreground': index() > 0 }}
              >
                <span class="shrink-0 font-mono text-grimorio-gold">{entry.roll}</span>
                <span class="min-w-0">{props.render(entry.result)}</span>
              </li>
            )}
          </For>
        </ul>
      </Show>
    </div>
  )
}

/**
 * Rooms decide everything else: the book's size band, how many threats to plan
 * (one per three rooms) and how long the place will take at the table.
 */
function DungeonPlanner() {
  const [rooms, setRooms] = createSignal(6)
  const [objective, setObjective] = createSignal('')
  const ideas = createRollHistory<DungeonIdea>()

  const size = createMemo<string | null>(() => classifyDungeonSize(Math.max(1, rooms())))
  const row = () => {
    const current = size()
    return current ? dungeonSizeRow(current) : null
  }

  return (
    <div class="grid gap-3 lg:grid-cols-2">
      <div class="space-y-3 rounded-sm border border-grimorio-iron p-3">
        <div class="flex flex-wrap items-end gap-3">
          <div class="space-y-1">
            <label
              for="dungeon-rooms"
              class="block font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground"
            >
              Salas
            </label>
            <NumberInput
              id="dungeon-rooms"
              min={1}
              max={60}
              value={rooms()}
              onChange={(value) => setRooms(Math.max(1, value))}
              class="w-24"
              aria-label="Número de salas"
              spinnerLabel="salas"
            />
          </div>
          <div class="min-w-48 flex-1 space-y-1">
            <label
              for="dungeon-objective"
              class="block font-heading text-[10px] uppercase tracking-[0.14em] text-muted-foreground"
            >
              Objetivo
            </label>
            <Input
              id="dungeon-objective"
              value={objective()}
              onInput={(event) => setObjective(event.currentTarget.value)}
              placeholder="Resgatar o alquimista"
            />
          </div>
        </div>

        <Show when={row()}>
          {(sizeRow) => (
            <p class="flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
              <span class="text-grimorio-gold">{SIZE_LABEL[sizeRow().size]}</span>
              <span>{plannedThreats(Math.max(1, rooms()))} ameaças</span>
              <span>{PACING_LABEL[sizeRow().pacing] ?? sizeRow().pacing}</span>
            </p>
          )}
        </Show>
      </div>

      <TableCard
        title="Ideia de masmorra"
        dice="d20"
        page="Tab 6-2"
        onRoll={() => {
          const roll = rollDice(1, 20)
          ideas.push(roll, dungeonIdeaFromRoll(roll))
        }}
        onClear={ideas.clear}
        entries={ideas.entries()}
        render={(idea) => idea.label}
      />
    </div>
  )
}
