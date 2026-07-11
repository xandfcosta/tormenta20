import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check, Sparkles, X } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { api, type Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import {
  useAllConditionals,
  type ConditionalEntry,
} from '@/entities/character/derived'
import { characterQueryOptions } from '@/entities/character/queries'
import { cn } from '@/shared/lib/utils'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import { getCatalogItem } from '@tormenta20/t20-data'
import type { ConditionalEffect, Modifier } from '@tormenta20/t20-data'
import {
  accentStrong,
  subtleText,
  surface,
} from '@/shared/lib/sheet-theme'
import { signed } from './signed'

/**
 * Effects tab — combines two categories:
 *
 *  1. `ActiveEffectsSection` — consumables that have been used and
 *     grant scene/day-scoped bonuses. Managed by the backend
 *     `/active-effects` endpoints; ending a scene/day clears the
 *     matching scope.
 *  2. `ConditionalsSection` — situational modifiers (terrain, target
 *     type, stance) that live on items or class powers. Toggled in
 *     the frontend via `useConditionalsStore`; state is per-character
 *     and persists in localStorage.
 *
 * A conditional whose `effect.flag` is set is grouped with siblings
 * carrying the same flag so multi-modifier stances (e.g. Fúria)
 * activate as a single row.
 */
export function EffectsPanel({ character }: { character: Character }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      <ActiveEffectsSection character={character} />
      <ConditionalsSection character={character} />
    </div>
  )
}

/**
 * Human-readable label for a Modifier target — used inside the
 * conditional rows so a player can read "Ataque +2" instead of
 * `{k:'attack', scope:'all'}`. Kept exhaustive so a missed case is a
 * TypeScript error rather than a silently-empty row.
 */
function describeConditionalTarget(target: Modifier['target']): string {
  switch (target.k) {
    case 'expertise':
      return target.name
    case 'expertiseAll':
      return 'todas perícias'
    case 'expertiseRemovePenalty':
      return `${target.name} (sem penalidade)`
    case 'expertiseByAttribute':
      return `Perícias de ${target.attribute}`
    case 'attribute':
      return target.name
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'limite de Des na Defesa'
    case 'resistance':
      return 'Resistência'
    case 'fearResistance':
      return 'Resistência (medo)'
    case 'attack':
      return target.scope === 'this' ? 'Ataque (esta arma)' : 'Ataque'
    case 'damage':
      return target.scope === 'this' ? 'Dano (esta arma)' : 'Dano'
    case 'critRange':
      return 'Margem de crítico'
    case 'critMult':
      return 'Multiplicador de crítico'
    case 'pmLimit':
      return 'Limite de PM'
    case 'pmCost':
      return 'Custo de PM'
    case 'catalyst':
      return `Catalisador ${target.school}`
    case 'spellDC':
      return 'CD Magia'
    case 'inventorySlots':
      return 'Espaços'
    case 'displacement':
      return 'Deslocamento'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias afetadas'
    case 'tempHp':
      return 'PV temp.'
    case 'tempMp':
      return 'PM temp.'
    case 'maneuver':
      return `Manobra: ${target.name}`
    case 'flag':
      return `Estado: ${target.name}`
  }
}

function ActiveEffectsSection({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const effects = character.activeEffects ?? []

  const remove = useMutation<{ id: number }, Error, number>({
    mutationFn: (effectId) =>
      api.characters.removeActiveEffect(character.id, effectId),
    onSuccess: ({ id }) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              activeEffects: prev.activeEffects.filter((e) => e.id !== id),
            }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })
  const endScene = useMutation<Character, Error, void>({
    mutationFn: () => api.characters.endScene(character.id),
    onSuccess: (next) => {
      qc.setQueryData<Character>(queryKey, next)
      invalidateCharacterDependents(qc, character.id)
    },
  })
  const endDay = useMutation<Character, Error, void>({
    mutationFn: () => api.characters.endDay(character.id),
    onSuccess: (next) => {
      qc.setQueryData<Character>(queryKey, next)
      invalidateCharacterDependents(qc, character.id)
    },
  })

  return (
    <section
      className={cn(
        'rounded-lg border p-3',
        surface,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>
          Efeitos consumíveis ativos
        </h3>
        <div className="flex gap-1">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-6 px-2 text-[11px]"
            onClick={() => endScene.mutate()}
            disabled={endScene.isPending}
            aria-label="Encerrar cena"
          >
            Encerrar cena
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="h-6 px-2 text-[11px]"
            onClick={() => {
              if (confirm('Encerrar dia? Limpa efeitos de cena e dia.'))
                endDay.mutate()
            }}
            disabled={endDay.isPending}
            aria-label="Encerrar dia"
          >
            Encerrar dia
          </Button>
        </div>
      </div>
      {effects.length === 0 ? (
        <p className={cn('mt-2 text-xs', subtleText)}>
          Nenhum consumível ativo. Use itens consumíveis no inventário.
        </p>
      ) : (
        <ul className="mt-2 space-y-1">
          {effects.map((eff) => (
            <ActiveEffectRow
              key={eff.id}
              effect={eff}
              onRemove={() => remove.mutate(eff.id)}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

function ActiveEffectRow({
  effect,
  onRemove,
}: {
  effect: Character['activeEffects'][number]
  onRemove: () => void
}) {
  const catalog = getCatalogItem(effect.catalogId)
  const name = catalog?.name ?? effect.catalogId
  return (
    <li
      className={cn(
        'flex items-center gap-2 rounded-md border px-2 py-1.5',
        'border-emerald-700/30 bg-emerald-50/60 dark:border-emerald-500/25 dark:bg-emerald-950/30',
      )}
    >
      <Sparkles className="size-3.5 shrink-0 text-emerald-700 dark:text-emerald-300" />
      <span className="flex-1 truncate text-sm text-zinc-800 dark:text-zinc-200">
        {name}
      </span>
      <span
        className={cn(
          'shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest',
          effect.scope === 'day'
            ? 'bg-amber-700/80 text-amber-50 dark:bg-amber-500/70 dark:text-zinc-900'
            : 'bg-emerald-700/80 text-emerald-50 dark:bg-emerald-500/70 dark:text-zinc-900',
        )}
      >
        {effect.scope === 'day' ? 'dia' : 'cena'}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
        onClick={onRemove}
        aria-label={`Remover ${name}`}
      >
        <X className="size-3.5" />
      </Button>
    </li>
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
 *  row, so e.g. all four Fúria modifiers activate together. */
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

function ConditionalsSection({ character }: { character: Character }) {
  const entries = useAllConditionals(character)
  const toggle = useConditionalsStore((s) => s.toggle)
  const setMany = useConditionalsStore((s) => s.setMany)
  const clear = useConditionalsStore((s) => s.clear)
  const activeCount = entries.filter((e) => e.active).length
  const groups = groupConditionals(entries)

  if (entries.length === 0) {
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

  return (
    <div
      className={cn(
        'flex min-h-0 flex-1 flex-col gap-2 overflow-hidden rounded-lg border p-3',
        surface,
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <p className={cn('text-xs', subtleText)}>
          {activeCount} de {entries.length} ativos — opt-in por cena/contexto
        </p>
        {activeCount > 0 && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-6 px-2 text-xs"
            onClick={() => clear(character.id)}
          >
            Limpar
          </Button>
        )}
      </div>
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
    </div>
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
        onClick={() => onToggle(!allActive)}
        className={cn(
          'flex w-full flex-col gap-1 rounded-md border px-2 py-1.5 text-left transition-colors',
          anyActive
            ? 'border-amber-700/50 bg-amber-100/70 dark:border-amber-500/40 dark:bg-amber-950/40'
            : 'border-amber-700/15 bg-zinc-100/40 hover:bg-amber-100/40 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:hover:bg-zinc-800/60',
        )}
      >
        <div className="flex items-center gap-3">
          <span
            className={cn(
              'flex size-5 shrink-0 items-center justify-center rounded border',
              allActive
                ? 'border-amber-700 bg-amber-700 text-amber-50 dark:border-amber-400 dark:bg-amber-400 dark:text-zinc-900'
                : anyActive
                  ? 'border-amber-700 bg-amber-100 dark:border-amber-400 dark:bg-amber-900/60'
                  : 'border-zinc-400 bg-transparent dark:border-zinc-600',
            )}
            aria-hidden
          >
            {allActive && <Check className="size-3" />}
          </span>
          <span className="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
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
        onClick={onToggle}
        className={cn(
          'flex w-full items-center gap-3 rounded-md border px-2 py-1.5 text-left transition-colors',
          active
            ? 'border-amber-700/50 bg-amber-100/70 dark:border-amber-500/40 dark:bg-amber-950/40'
            : 'border-amber-700/15 bg-zinc-100/40 hover:bg-amber-100/40 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:hover:bg-zinc-800/60',
        )}
      >
        <span
          className={cn(
            'flex size-5 shrink-0 items-center justify-center rounded border',
            active
              ? 'border-amber-700 bg-amber-700 text-amber-50 dark:border-amber-400 dark:bg-amber-400 dark:text-zinc-900'
              : 'border-zinc-400 bg-transparent dark:border-zinc-600',
          )}
          aria-hidden
        >
          {active && <Check className="size-3" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium text-zinc-800 dark:text-zinc-200">
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
