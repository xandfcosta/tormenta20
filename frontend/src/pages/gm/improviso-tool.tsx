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
import { SectionLabel, SectionTitle } from '@/shared/ui/section-label'

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
      <SectionTitle
        id="mesa-improviso">
        Improviso
      </SectionTitle>

      <div class="min-h-0 flex-1 space-y-4 overflow-y-auto pr-1">
        <div>
          <SectionLabel class="mb-2">
            Tabelas do Cap 6
          </SectionLabel>
          <div class="grid items-start gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <RuinaTable />
            <ChaseTable />
            <RewardTable />
          </div>
        </div>

        <div class="border-t border-grimorio-iron pt-4">
          <SectionLabel class="mb-2">
            Masmorra
          </SectionLabel>
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
        <SectionLabel tom="gold">
          {props.title}
        </SectionLabel>
        <p class="font-mono text-3xs text-muted-foreground">
          {props.dice} · {props.page}
        </p>
      </div>

      {/* O RESULTADO é o maior elemento do cartão, e isso é a issue inteira
          (ALE-170): esta ferramenta existe para produzir uma linha, e essa
          linha era 11px — menor que o botão que a pede, menor que o título,
          o menor elemento da tela. Uma tabela de improviso boa se lê do outro
          lado da mesa, porque quem rola é o mestre e quem ouve é a mesa.

          O histórico continua pequeno de propósito: ele é referência, não
          resposta. Antes os dois tinham o mesmo tamanho e a última rolagem se
          perdia entre as anteriores. */}
      <Show
        when={props.entries[0]}
        fallback={
          <p class="py-2 text-sm text-muted-foreground">Ainda não rolado.</p>
        }
      >
        {(ultima) => (
          <div class="flex items-baseline gap-3 py-1">
            <span class="shrink-0 font-mono text-3xl leading-none text-grimorio-gold tabular-nums">
              {ultima().roll}
            </span>
            <span class="min-w-0 text-base leading-snug text-foreground">
              {props.render(ultima().result)}
            </span>
          </div>
        )}
      </Show>

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

      <Show when={props.entries.length > 1}>
        <ul class="space-y-0.5 border-t border-grimorio-iron/40 pt-1.5">
          <For each={props.entries.slice(1)}>
            {(entry) => (
              <li class="flex gap-2 text-2xs text-muted-foreground">
                <span class="shrink-0 font-mono">{entry.roll}</span>
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
            <SectionLabel
              for="dungeon-rooms"
             
             as="label" class="text-3xs block">
              Salas
            </SectionLabel>
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
            <SectionLabel
              for="dungeon-objective"
             
             as="label" class="text-3xs block">
              Objetivo
            </SectionLabel>
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
            <p class="flex flex-wrap gap-x-3 gap-y-1 text-2xs text-muted-foreground">
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
