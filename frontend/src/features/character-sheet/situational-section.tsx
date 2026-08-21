import { Check, Lock } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import { type ConditionalEntry, allConditionals } from '@/entities/character/derived'
import { type ItemFlagEffect, equippedItemFlagEffects } from '@/entities/character/effect-source'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import { type ConditionalGroup, situationalGroups } from './conditional-groups'
import { describeConditionalTarget } from './conditional-target-label'
import { signed } from './signed'

/**
 * "Situação" — the opt-in modifiers that depend on context (terrain, target
 * type, homebrew item toggles). Client state, unlike the effects above: they
 * live in the conditionals store and the sheet recomputes against whatever is
 * switched on.
 *
 * Always-on item flags (heavy armor's "Fadiga ao dormir") have no toggle — they
 * render read-only, so this block never contradicts the header warnings that
 * surface the same flags.
 */
export function SituationalSection(props: { character: Character }) {
  const conditionals = useConditionals()
  const groups = createMemo(() =>
    situationalGroups(allConditionals(props.character, conditionals.active(props.character.id))),
  )
  const flagEffects = createMemo(() => equippedItemFlagEffects(props.character.items))

  return (
    <Show
      when={groups().length > 0 || flagEffects().length > 0}
      fallback={
        <p class="rounded-none border border-grimorio-iron p-6 text-center text-sm text-muted-foreground">
          Nenhum efeito condicional disponível. Equipe itens com modificadores situacionais
          (terreno, contexto, contra alvo) para vê-los aqui.
        </p>
      }
    >
      <section class="space-y-2 rounded-none border border-grimorio-iron p-3">
        <ItemFlagList effects={flagEffects()} />
        <ToggleableConditionals character={props.character} groups={groups()} />
      </section>
    </Show>
  )
}

/** Read-only rows for always-on item flags — no switch, only provenance. */
function ItemFlagList(props: { effects: ItemFlagEffect[] }) {
  return (
    <Show when={props.effects.length > 0}>
      <div class="space-y-1">
        <p class="text-3xs font-bold uppercase tracking-widest text-muted-foreground">
          Sempre ativos (itens equipados)
        </p>
        <ul class="space-y-1">
          <For each={props.effects}>
            {(effect) => (
              <li class="flex items-center gap-2 rounded-sm border border-border bg-muted px-2 py-1.5">
                <Lock aria-hidden="true" class="size-3.5 shrink-0 text-muted-foreground" />
                <span class="min-w-0 flex-1 truncate text-sm text-foreground">{effect.label}</span>
                <span class="shrink-0 truncate text-2xs text-muted-foreground">
                  {effect.source}
                </span>
              </li>
            )}
          </For>
        </ul>
      </div>
    </Show>
  )
}

function ToggleableConditionals(props: { character: Character; groups: ConditionalGroup[] }) {
  const conditionals = useConditionals()
  const shownEntries = createMemo(() =>
    props.groups.flatMap((group) => (group.kind === 'single' ? [group.entry] : group.entries)),
  )
  const activeCount = createMemo(() => shownEntries().filter((entry) => entry.active).length)

  // NOT the store's `clear`: that would also switch off active power stances,
  // which end only through their own Encerrar in the Poderes block.
  const clearShown = () =>
    conditionals.setMany(
      props.character.id,
      shownEntries().map((entry) => entry.id),
      false,
    )

  return (
    <Show when={props.groups.length > 0}>
      <div class="flex items-center justify-between gap-2">
        <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
          Situação — opt-in por contexto
        </h3>
        <Show when={activeCount() > 0}>
          <Button type="button" variant="ghost" size="sm" class="h-6 px-2 text-xs" onClick={clearShown}>
            Limpar
          </Button>
        </Show>
      </div>
      <p class="text-xs text-muted-foreground">
        {activeCount()} de {shownEntries().length} ativos
      </p>
      <ul class="space-y-1">
        <For each={props.groups}>
          {(group) =>
            group.kind === 'single' ? (
              <ConditionalRow
                entry={group.entry}
                onToggle={() => conditionals.toggle(props.character.id, group.entry.id)}
              />
            ) : (
              <FlagGroupRow
                group={group}
                onToggle={(value) =>
                  conditionals.setMany(
                    props.character.id,
                    group.entries.map((entry) => entry.id),
                    value,
                  )
                }
              />
            )
          }
        </For>
      </ul>
    </Show>
  )
}

const ROW_CLASS =
  'flex w-full rounded-sm border border-border bg-muted px-2 py-1.5 text-left transition-colors hover:bg-accent'

/** Sign-colored amount, shared by both row kinds. */
function Amount(props: { amount: number; suffix?: string }) {
  return (
    <span
      class={cn(
        'shrink-0 font-mono font-semibold',
        props.amount >= 0 ? 'text-emerald-300' : 'text-red-300',
      )}
    >
      {signed(props.amount)}
      {props.suffix ? ` ${props.suffix}` : ''}
    </span>
  )
}

function Checkbox(props: { checked: boolean }) {
  return (
    <span
      aria-hidden="true"
      class={cn(
        'flex size-5 shrink-0 items-center justify-center rounded-md border border-border',
        props.checked ? 'bg-muted text-foreground' : 'bg-transparent',
      )}
    >
      <Show when={props.checked}>
        <Check class="size-3" />
      </Show>
    </span>
  )
}

function FlagGroupRow(props: {
  group: Extract<ConditionalGroup, { kind: 'flag' }>
  onToggle: (value: boolean) => void
}) {
  const allActive = () => props.group.entries.every((entry) => entry.active)
  return (
    <li>
      <button
        type="button"
        role="switch"
        aria-checked={allActive()}
        onClick={() => props.onToggle(!allActive())}
        class={cn(ROW_CLASS, 'flex-col gap-1')}
      >
        <div class="flex w-full items-center gap-3">
          <Checkbox checked={allActive()} />
          <span class="truncate text-sm font-medium text-foreground">{props.group.source}</span>
          <span class="ml-auto truncate text-2xs text-muted-foreground">
            {props.group.label}
          </span>
        </div>
        <ul class="ml-8 w-full space-y-0.5 text-2xs">
          <For each={props.group.entries}>
            {(entry) => (
              <li class="flex items-center justify-between gap-2">
                <span class="truncate text-muted-foreground">
                  {describeConditionalTarget(entry.effect.target)}
                </span>
                <Amount amount={entry.effect.amount} />
              </li>
            )}
          </For>
        </ul>
      </button>
    </li>
  )
}

function ConditionalRow(props: { entry: ConditionalEntry; onToggle: () => void }) {
  return (
    <li>
      <button
        type="button"
        role="switch"
        aria-checked={props.entry.active}
        onClick={() => props.onToggle()}
        class={cn(ROW_CLASS, 'items-center gap-3')}
      >
        <Checkbox checked={props.entry.active} />
        <div class="min-w-0 flex-1">
          <div class="flex items-center justify-between gap-2">
            <span class="truncate text-sm font-medium text-foreground">
              {props.entry.effect.source}
            </span>
            <Amount
              amount={props.entry.effect.amount}
              suffix={describeConditionalTarget(props.entry.effect.target)}
            />
          </div>
          <p class="truncate text-2xs text-muted-foreground">{props.entry.effect.note}</p>
        </div>
      </button>
    </li>
  )
}
