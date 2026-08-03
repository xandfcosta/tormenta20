import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Plus, Search, Sparkles, X } from 'lucide-react'
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
import { parseEffectModifiers } from '@/entities/character/derived'
import {
  effectSourceFacts,
  effectSourceName,
} from '@/entities/character/effect-source'
import { characterQueryOptions } from '@/entities/character/queries'
import { applyPoolResult } from '@/entities/character/temp-hp-pool'
import { cn } from '@/shared/lib/utils'
import { usePowerUsesStore } from '@/shared/stores/power-uses-store'
import { SPELL_CATALOG } from '@tormenta20/t20-data'
import { accentStrong, subtleText, surface } from '@/shared/lib/sheet-theme'
import { describeConditionalTarget } from './conditional-target-label'
import { FactChips } from './fact-chips'
import { normalize } from './normalize'
import { signed } from './signed'

/**
 * "Efeitos ativos" — consumables/spell buffs that have been used and grant
 * scene/day-scoped bonuses. Managed by the backend `/active-effects`
 * endpoints; ending a scene/day clears the matching scope (and resets the
 * local limited-power-use counters). Split out of effects-panel.tsx so each
 * Efeitos category owns a module.
 */
export function ActiveEffectsSection({ character }: { character: Character }) {
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
  // Scene/day boundaries also reset the local limited-power-use counters
  // (Poderes tab "usado X/1 cena|dia").
  const resetSceneUses = usePowerUsesStore((s) => s.resetScene)
  const resetDayUses = usePowerUsesStore((s) => s.resetDay)
  const endScene = useMutation({
    mutationFn: () => api.characters.endScene(character.id),
    onSuccess: (cleared) => {
      dropClearedScopes(cleared)
      resetSceneUses(character.id)
    },
  })
  const endDay = useMutation({
    mutationFn: () => api.characters.endDay(character.id),
    onSuccess: (cleared) => {
      dropClearedScopes(cleared)
      resetDayUses(character.id)
    },
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
          <ConfirmDialog
            title="Encerrar cena?"
            description="Limpa todos os efeitos de cena (buffs, poções ativas)."
            confirmLabel="Encerrar cena"
            onConfirm={() => endScene.mutate()}
            trigger={
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-6 px-2 text-[11px]"
                disabled={endScene.isPending}
                aria-label="Encerrar cena"
              >
                Encerrar cena
              </Button>
            }
          />
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
    // Spell buffs always come back as a plain row; `applyPoolResult` also
    // covers the temp-PV pool outcomes the shared endpoint can produce.
    onSuccess: (result) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? applyPoolResult(prev, result) : prev,
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
