import { ConditionsSection } from './conditions-section'
import { Check, Lock } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import type { Character } from '@/shared/api/api'
import {
  useAllConditionals,
  type ConditionalEntry,
} from '@/entities/character/derived'
import {
  equippedItemFlagEffects,
  type ItemFlagEffect,
} from '@/entities/character/effect-source'
import { cn } from '@/shared/lib/utils'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import { FLAG_ACTIVATIONS } from '@tormenta20/t20-data'
import type { ConditionalEffect } from '@tormenta20/t20-data'
import {
  accentStrong,
  subtleText,
  surface,
} from '@/shared/lib/sheet-theme'
import { ActiveEffectsSection } from './active-effects-section'
import { describeConditionalTarget } from './conditional-target-label'
import { signed } from './signed'
import { StancesSection } from './stances-section'

/**
 * Effects tab — combines three categories:
 *
 *  1. `StancesSection` — power stances (FLAG_ACTIVATIONS) that are ON.
 *     Their on-switch moved to the Poderes tab (Phase 3); here only the
 *     active card + Encerrar renders. OFF stances don't appear at all.
 *  2. `ActiveEffectsSection` — consumables that have been used and
 *     grant scene/day-scoped bonuses. Managed by the backend
 *     `/active-effects` endpoints; ending a scene/day clears the
 *     matching scope.
 *  3. `ConditionalsSection` ("Situação") — situational modifiers
 *     (terrain, target type, homebrew item toggles) that live on items
 *     or class powers. Toggled in the frontend via
 *     `useConditionalsStore`; state is per-character and persists in
 *     localStorage.
 *
 * An item-sourced conditional whose `effect.flag` is set (homebrew-*)
 * is grouped with siblings carrying the same flag so multi-modifier
 * groups activate as a single row.
 *
 * Always-on flag effects from equipped items (heavy armor's "Fadiga ao
 * dormir", "Armadura pesada") have no toggle — they render as a
 * read-only group inside the conditionals section so the tab never
 * contradicts the header warnings that surface the same flags.
 */
export function EffectsPanel({ character }: { character: Character }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      <ConditionsSection character={character} />
      <StancesSection character={character} />
      <ActiveEffectsSection character={character} />
      <ConditionalsSection character={character} />
    </div>
  )
}

type ConditionalGroup =
  | { kind: 'single'; entry: ConditionalEntry }
  | {
      kind: 'flag'
      flag: string
      label: string
      source: string
      entries: ConditionalEntry[]
    }

/** Folds individual conditional entries that share a `flag` into one toggle
 *  row, so e.g. a homebrew item's multi-modifier group activates together. */
function groupConditionals(entries: ConditionalEntry[]): ConditionalGroup[] {
  const byFlag = new Map<string, ConditionalEntry[]>()
  const groups: ConditionalGroup[] = []
  for (const e of entries) {
    const f = e.effect.flag
    if (!f) {
      groups.push({ kind: 'single', entry: e })
      continue
    }
    const arr = byFlag.get(f) ?? []
    arr.push(e)
    byFlag.set(f, arr)
  }
  for (const [flag, list] of byFlag) {
    groups.push({
      kind: 'flag',
      flag,
      label: list[0].effect.note,
      source: list[0].effect.source,
      entries: list,
    })
  }
  return groups
}

/**
 * Split decision (Phase 3): only power stances registered in FLAG_ACTIVATIONS
 * moved to the Poderes tab — item-sourced flag groups (homebrew-*) and every
 * non-flag conditional keep their toggle here in Situação.
 */
function situationalGroups(entries: ConditionalEntry[]): ConditionalGroup[] {
  return groupConditionals(entries).filter(
    (g) => g.kind === 'single' || !FLAG_ACTIVATIONS[g.flag],
  )
}

function ConditionalsSection({ character }: { character: Character }) {
  const entries = useAllConditionals(character)
  const groups = situationalGroups(entries)
  const flagEffects = equippedItemFlagEffects(character.items)

  if (groups.length === 0 && flagEffects.length === 0) {
    return <ConditionalsEmptyState />
  }

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col gap-2 overflow-hidden rounded-lg border p-3',
        surface,
      )}
    >
      <ItemFlagList effects={flagEffects} />
      <ToggleableConditionals character={character} groups={groups} />
    </div>
  )
}

function ConditionalsEmptyState() {
  return (
    <div
      className={cn(
        'flex flex-1 items-center justify-center rounded-lg border p-6 text-center text-sm',
        surface,
        subtleText,
      )}
    >
      Nenhum efeito condicional disponível. Equipe itens com modificadores
      situacionais (terreno, contexto, contra alvo) para vê-los aqui.
    </div>
  )
}

/** Read-only rows for always-on item flags — no switch, only provenance. */
function ItemFlagList({ effects }: { effects: ItemFlagEffect[] }) {
  if (effects.length === 0) return null
  return (
    <div className="space-y-1">
      <p className={cn('text-[10px] font-bold uppercase tracking-widest', subtleText)}>
        Sempre ativos (itens equipados)
      </p>
      <ul className="space-y-1">
        {effects.map((e) => (
          <li
            key={`${e.flag}:${e.source}`}
            className="flex items-center gap-2 rounded-md border border-border bg-muted px-2 py-1.5"
          >
            <Lock className={cn('size-3.5 shrink-0', subtleText)} aria-hidden />
            <span className="min-w-0 flex-1 truncate text-sm text-foreground">
              {e.label}
            </span>
            <span className={cn('shrink-0 truncate text-[11px]', subtleText)}>
              {e.source}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

function ToggleableConditionals({
  character,
  groups,
}: {
  character: Character
  groups: ConditionalGroup[]
}) {
  const toggle = useConditionalsStore((s) => s.toggle)
  const setMany = useConditionalsStore((s) => s.setMany)
  // PM debit for power stances moved to usePowerAction.activateStance (Phase
  // 3) — the groups reaching here are homebrew item toggles, always free.
  const shownEntries = groups.flatMap((g) =>
    g.kind === 'single' ? [g.entry] : g.entries,
  )
  if (groups.length === 0) return null
  const activeCount = shownEntries.filter((e) => e.active).length
  const clearShown = () =>
    // Not the store's `clear`: that would also switch off active power
    // stances, which now only end via their Encerrar button.
    setMany(
      character.id,
      shownEntries.map((e) => e.id),
      false,
    )
  return (
    <>
      <div className="flex items-center justify-between gap-2">
        <h3 className={cn('text-sm font-bold', accentStrong)}>
          Situação — opt-in por contexto
        </h3>
        {activeCount > 0 && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={clearShown}
          >
            Limpar
          </Button>
        )}
      </div>
      <p className={cn('text-xs', subtleText)}>
        {activeCount} de {shownEntries.length} ativos
      </p>
      <ul className="flex-1 space-y-1 overflow-y-auto pr-1">
        {groups.map((g) =>
          g.kind === 'single' ? (
            <ConditionalRow
              key={g.entry.id}
              entry={g.entry}
              onToggle={() => toggle(character.id, g.entry.id)}
            />
          ) : (
            <FlagGroupRow
              key={g.flag}
              group={g}
              onToggle={(value) =>
                setMany(
                  character.id,
                  g.entries.map((e) => e.id),
                  value,
                )
              }
            />
          ),
        )}
      </ul>
    </>
  )
}

function FlagGroupRow({
  group,
  onToggle,
}: {
  group: Extract<ConditionalGroup, { kind: 'flag' }>
  onToggle: (value: boolean) => void
}) {
  const allActive = group.entries.every((e) => e.active)
  const anyActive = group.entries.some((e) => e.active)
  return (
    <li>
      <button
        type="button"
        role="switch"
        aria-checked={allActive}
        onClick={() => onToggle(!allActive)}
        className={cn(
          'flex w-full flex-col gap-1 rounded-md border px-2 py-1.5 text-left transition-colors',
          anyActive
            ? 'border-border bg-muted  '
            : 'border-border bg-muted hover:bg-muted   dark:hover:bg-muted',
        )}
      >
        <div className="flex items-center gap-3">
          <span
            className={cn(
              'flex size-5 shrink-0 items-center justify-center rounded border',
              allActive
                ? 'border-border bg-muted text-foreground   '
                : anyActive
                  ? 'border-border bg-muted  '
                  : 'border-border bg-transparent ',
            )}
            aria-hidden
          >
            {allActive && <Check className="size-3" />}
          </span>
          <span className="truncate text-sm font-medium text-foreground ">
            {group.source}
          </span>
          <span className={cn('ml-auto truncate text-[11px]', subtleText)}>
            {group.label}
          </span>
        </div>
        <ul className="ml-8 space-y-0.5 text-[11px]">
          {group.entries.map((e) => (
            <li key={e.id} className="flex items-center justify-between gap-2">
              <span className={cn('truncate', subtleText)}>
                {describeConditionalTarget(e.effect.target)}
              </span>
              <span
                className={cn(
                  'shrink-0 font-mono font-semibold',
                  e.effect.amount >= 0
                    ? 'text-emerald-700 dark:text-emerald-300'
                    : 'text-red-700 dark:text-red-300',
                )}
              >
                {signed(e.effect.amount)}
              </span>
            </li>
          ))}
        </ul>
      </button>
    </li>
  )
}

function ConditionalRow({
  entry,
  onToggle,
}: {
  entry: { id: string; effect: ConditionalEffect; active: boolean }
  onToggle: () => void
}) {
  const { effect, active } = entry
  const targetLabel = describeConditionalTarget(effect.target)
  return (
    <li>
      <button
        type="button"
        role="switch"
        aria-checked={active}
        onClick={onToggle}
        className={cn(
          'flex w-full items-center gap-3 rounded-md border px-2 py-1.5 text-left transition-colors',
          active
            ? 'border-border bg-muted  '
            : 'border-border bg-muted hover:bg-muted   dark:hover:bg-muted',
        )}
      >
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded border',
            active
              ? 'border-border bg-muted text-foreground   '
              : 'border-border bg-transparent ',
          )}
          aria-hidden
        >
          {active && <Check className="size-3" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium text-foreground ">
              {effect.source}
            </span>
            <span
              className={cn(
                'shrink-0 font-mono text-sm font-semibold',
                effect.amount >= 0
                  ? 'text-emerald-700 dark:text-emerald-300'
                  : 'text-red-700 dark:text-red-300',
              )}
            >
              {signed(effect.amount)} {targetLabel}
            </span>
          </div>
          <p className={cn('truncate text-[11px]', subtleText)}>
            {effect.note}
          </p>
        </div>
      </button>
    </li>
  )
}
