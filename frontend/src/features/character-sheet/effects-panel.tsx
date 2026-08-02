import { ConditionsSection } from './conditions-section'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Plus, Search, Sparkles, X } from 'lucide-react'
import { Check } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { api, type Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import {
  parseEffectModifiers,
  useAllConditionals,
  type ConditionalEntry,
} from '@/entities/character/derived'
import {
  effectSourceFacts,
  effectSourceName,
} from '@/entities/character/effect-source'
import { characterQueryOptions } from '@/entities/character/queries'
import { cn } from '@/shared/lib/utils'
import { useConditionalsStore } from '@/shared/stores/conditionals-store'
import { SPELL_CATALOG } from '@tormenta20/t20-data'
import type { ConditionalEffect, Modifier } from '@tormenta20/t20-data'
import {
  accentStrong,
  subtleText,
  surface,
} from '@/shared/lib/sheet-theme'
import { FactChips } from './fact-chips'
import { normalize } from './normalize'
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
      <ConditionsSection character={character} />
      <ActiveEffectsSection character={character} />
      <ConditionalsSection character={character} />
    </div>
  )
}

// Attribute keys arrive raw (e.g. 'charisma') on catalog modifiers — map to
// the pt-BR name so a row reads "Carisma +2" instead of "charisma +2".
const ATTRIBUTE_PT: Record<string, string> = {
  strength: 'Força',
  dexterity: 'Destreza',
  constitution: 'Constituição',
  intelligence: 'Inteligência',
  wisdom: 'Sabedoria',
  charisma: 'Carisma',
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
      return ATTRIBUTE_PT[target.name] ?? target.name
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
    case 'flySpeed':
      return 'Voo'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias afetadas'
    case 'tempHp':
      return 'PV temp.'
    case 'tempMp':
      return 'PM temp.'
    case 'maxPv':
      return 'PV máximo'
    case 'maxPm':
      return 'PM máximo'
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
  // Delta merge: drop the cached effects whose scope the server cleared.
  const dropClearedScopes = ({ clearedScopes }: { clearedScopes: string[] }) => {
    qc.setQueryData<Character>(queryKey, (prev) =>
      prev
        ? {
            ...prev,
            activeEffects: prev.activeEffects.filter(
              (e) => !clearedScopes.includes(e.scope),
            ),
          }
        : prev,
    )
    invalidateCharacterDependents(qc, character.id)
  }
  const endScene = useMutation({
    mutationFn: () => api.characters.endScene(character.id),
    onSuccess: dropClearedScopes,
  })
  const endDay = useMutation({
    mutationFn: () => api.characters.endDay(character.id),
    onSuccess: dropClearedScopes,
  })

  return (
    <section
      className={cn(
        'rounded-lg border p-3',
        surface,
      )}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className={cn('text-sm font-bold', accentStrong)}>
          Efeitos ativos
        </h3>
        <div className="flex flex-wrap gap-1">
          <ApplyEffectDialog character={character} />
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
          <ConfirmDialog
            title="Encerrar dia?"
            description="Limpa efeitos de cena e dia."
            confirmLabel="Encerrar dia"
            onConfirm={() => endDay.mutate()}
            trigger={
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-6 px-2 text-[11px]"
                disabled={endDay.isPending}
                aria-label="Encerrar dia"
              >
                Encerrar dia
              </Button>
            }
          />
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
  // Name + modifiers resolve for BOTH item and spell sources: the name via the
  // spell-aware resolver, the modifiers from the effect's own persisted blob
  // (works regardless of source, unlike reading `catalog.consumable.modifiers`).
  const name = effectSourceName(effect.catalogId)
  const modifiers = parseEffectModifiers(effect.modifiers)
  const facts = effectSourceFacts(effect.catalogId)
  return (
    <li
      className={cn(
        'rounded-md border px-2 py-1.5',
        'border-emerald-700/30 bg-emerald-50/60 dark:border-emerald-500/25 dark:bg-emerald-950/30',
      )}
    >
      <div className="flex items-center gap-2">
        <Sparkles className="size-3.5 shrink-0 text-emerald-700 dark:text-emerald-300" />
        <span className="flex-1 truncate text-sm text-foreground">{name}</span>
        <span
          className={cn(
            'shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-widest',
            effect.scope === 'day'
              ? 'bg-muted text-foreground'
              : 'bg-emerald-700/80 text-emerald-50 dark:bg-emerald-500/70',
          )}
        >
          {effect.scope === 'day' ? 'dia' : 'cena'}
        </span>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-foreground hover:bg-red-100 hover:text-red-700 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={onRemove}
          aria-label={`Remover ${name}`}
        >
          <X className="size-3.5" />
        </Button>
      </div>
      {modifiers.length > 0 && (
        <ul className="ml-5 mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-[11px]">
          {modifiers.map((m, i) => (
            <li key={i} className="flex items-center gap-1">
              <span className={subtleText}>
                {describeConditionalTarget(m.target)}
              </span>
              <span
                className={cn(
                  'font-mono font-semibold',
                  m.amount >= 0
                    ? 'text-emerald-700 dark:text-emerald-300'
                    : 'text-red-700 dark:text-red-300',
                )}
              >
                {signed(m.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <FactChips facts={facts} className="ml-5 mt-1" />
      {modifiers.length === 0 && facts.length === 0 && (
        <p className={cn('ml-5 mt-1 text-[11px] italic', subtleText)}>
          Sem efeito mecânico
        </p>
      )}
    </li>
  )
}

/** Spell buffs the player can self-apply, resolved once from the catalog. A
 *  spell qualifies only if it carries a Phase-1 `SpellBuff` block. */
const BUFF_SPELLS = Object.values(SPELL_CATALOG).filter((s) => s.buff)

/**
 * Manual self-apply of a spell buff as a scene/day ActiveEffect. Buffs are
 * never auto-applied from another caster — the target (or GM) picks the source
 * and applies it here. Scope defaults to the buff's `defaultScope`.
 */
function ApplyEffectDialog({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const apply = useMutation({
    mutationFn: (spellId: string) =>
      api.characters.applyEffect(character.id, { spellId }),
    // Delta merge: upsert the returned ActiveEffect row (replace any existing
    // one for the same spell+scope, since the server upserts on that key).
    onSuccess: (effect) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? {
              ...prev,
              activeEffects: [
                ...prev.activeEffects.filter(
                  (e) =>
                    e.id !== effect.id &&
                    !(
                      e.catalogId === effect.catalogId &&
                      e.scope === effect.scope
                    ),
                ),
                effect,
              ],
            }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
      setOpen(false)
    },
  })

  const matches = useMemo(() => {
    const q = normalize(query.trim())
    if (!q) return BUFF_SPELLS
    return BUFF_SPELLS.filter((s) => normalize(s.name).includes(q))
  }, [query])

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-6 px-2 text-[11px]"
        >
          <Plus className="mr-1 size-3" />
          Aplicar efeito
        </Button>
      </DialogTrigger>
      <DialogContent className="flex max-h-[80vh] flex-col gap-3">
        <DialogHeader>
          <DialogTitle>Aplicar efeito de magia</DialogTitle>
        </DialogHeader>
        <div className="relative">
          <Search className="absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar magia…"
            className="pl-8"
          />
        </div>
        {matches.length === 0 ? (
          <p className={cn('py-6 text-center text-sm', subtleText)}>
            Nenhuma magia com efeito aplicável.
          </p>
        ) : (
          <VirtualList
            items={matches}
            estimateSize={60}
            gap={4}
            getKey={(spell) => spell.id}
            className="min-h-0 flex-1"
            renderItem={(spell) => (
              <button
                type="button"
                disabled={apply.isPending}
                onClick={() => apply.mutate(spell.id)}
                className="flex w-full flex-col gap-1 rounded-md border p-2 text-left transition-colors hover:bg-muted disabled:opacity-50"
              >
                <div className="flex items-center justify-between gap-2">
                  <span className="truncate text-sm font-medium text-foreground">
                    {spell.name}
                  </span>
                  <span className={cn('shrink-0 text-[10px] uppercase', subtleText)}>
                    {spell.buff?.defaultScope === 'day' ? 'dia' : 'cena'}
                  </span>
                </div>
                <ul className="flex flex-wrap gap-x-3 gap-y-0.5 text-[11px]">
                  {(spell.buff?.modifiers ?? []).map((m, i) => (
                    <li key={i} className="flex items-center gap-1">
                      <span className={subtleText}>
                        {describeConditionalTarget(m.target)}
                      </span>
                      <span
                        className={cn(
                          'font-mono font-semibold',
                          m.amount >= 0
                            ? 'text-emerald-700 dark:text-emerald-300'
                            : 'text-red-700 dark:text-red-300',
                        )}
                      >
                        {signed(m.amount)}
                      </span>
                    </li>
                  ))}
                </ul>
                <FactChips facts={spell.buff?.facts ?? []} />
              </button>
            )}
          />
        )}
      </DialogContent>
    </Dialog>
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
