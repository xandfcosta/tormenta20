import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Sparkles, Zap } from 'lucide-react'
import type { CatalogSpell } from '@tormenta20/t20-data'
import {
  SPELLCASTER_CLASSES,
  highestCircleAtLevel,
  SPELL_BASE_PM_COST,
  firstErrorMessage,
  validateCast,
} from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { NumberInput } from '@/shared/ui/number-input'
import { ApiError, api } from '@/shared/api/api'
import type { CastSpellResult, Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import { characterQueryOptions } from '@/entities/character/queries'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

type AugmentPick = { augmentIndex: number; stacks: number }

/**
 * Cast dialog — user picks stacks per augment (0 = not taken; `muda`
 * augments capped at 1). Shows a live PM total + per-spell limit
 * check. Server is authoritative — the client-side preview is only a
 * UX hint.
 *
 * `compact` renders the trigger icon-only below `sm` (row headers at
 * phone width) while keeping the labeled button on larger screens; the
 * aria-label carries the spell name either way.
 */
export function CastSpellDialog({
  spell,
  character,
  disabled,
  compact = false,
}: {
  spell: CatalogSpell
  character: Character
  disabled?: boolean
  compact?: boolean
}) {
  const [open, setOpen] = useState(false)
  const [stacksByIndex, setStacksByIndex] = useState<Map<number, number>>(
    new Map(),
  )
  const [error, setError] = useState<string | null>(null)

  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const sheet = useComputedSheet(character)

  /**
   * Highest circle this character can CAST — gates aprimoramentos with
   * `requiresCircle`. A power-granted spell on a non-caster (Bárbaro com
   * Totem) is castable at its own circle only, so 2º+ upgrades lock
   * (p42/p171: sem acesso a círculos maiores, sem aprimoramentos deles).
   */
  const castableCircle = useMemo(() => {
    const casters = character.classes.filter((c) =>
      (SPELLCASTER_CLASSES as readonly string[]).includes(c.className),
    )
    const best = casters.reduce(
      (top, c) =>
        Math.max(
          top,
          highestCircleAtLevel(
            c.className as (typeof SPELLCASTER_CLASSES)[number],
            c.level,
          ),
        ),
      0,
    )
    return Math.max(best, spell.circle)
  }, [character.classes, spell.circle])

  const augmentPicks: AugmentPick[] = useMemo(() => {
    const out: AugmentPick[] = []
    for (const [augmentIndex, stacks] of stacksByIndex) {
      if (stacks > 0) out.push({ augmentIndex, stacks })
    }
    return out
  }, [stacksByIndex])

  const augmentPm = useMemo(
    () =>
      augmentPicks.reduce(
        (sum, p) => sum + spell.augments[p.augmentIndex].pmCost * p.stacks,
        0,
      ),
    [augmentPicks, spell.augments],
  )
  const basePm = SPELL_BASE_PM_COST[spell.circle]
  const totalPm = spell.circle === 0 ? 0 : basePm + augmentPm
  // Same number as the Limite PM HUD box — caster-class level (PDF p224) plus
  // pmLimit item bonuses — so the cast gate and the sheet never disagree.
  const perSpellLimit = sheet.pmLimit.total
  // Single source of truth for the cast preconditions (shared with the
  // backend). Prep-requirement stays server-enforced — the cast button only
  // shows for learned spells, and detecting the caster's prep rule client-side
  // isn't needed to predict PM outcomes.
  const castBlocked = firstErrorMessage(
    validateCast({
      circle: spell.circle,
      totalPm,
      pmLimit: perSpellLimit,
      mpCurrent: character.mpCurrent,
      needsPrep: false,
      prepared: true,
    }),
  )

  const cast = useMutation<CastSpellResult, Error, void, { prev?: Character }>({
    mutationFn: () =>
      api.characters.castSpell(character.id, spell.id, augmentPicks),
    onMutate: async () => {
      // Optimistic PM spend — validated above, so the server should agree.
      await qc.cancelQueries({ queryKey })
      const prev = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (c) =>
        c ? { ...c, mpCurrent: Math.max(0, c.mpCurrent - totalPm) } : c,
      )
      return { prev }
    },
    // Delta merge: authoritative PM + drop any catalyst effect the cast consumed.
    onSuccess: (delta) => {
      qc.setQueryData<Character>(queryKey, (c) =>
        c
          ? {
              ...c,
              mpCurrent: delta.mpCurrent,
              activeEffects: c.activeEffects.filter(
                (e) => !delta.removedEffectIds.includes(e.id),
              ),
            }
          : c,
      )
      invalidateCharacterDependents(qc, character.id)
      setOpen(false)
      setStacksByIndex(new Map())
      setError(null)
    },
    onError: (e, _v, ctx) => {
      if (ctx?.prev) qc.setQueryData(queryKey, ctx.prev)
      setError(e instanceof ApiError ? e.message : 'Erro ao conjurar')
    },
  })

  const setStacks = (index: number, next: number) => {
    setStacksByIndex((prev) => {
      const map = new Map(prev)
      if (next <= 0) map.delete(index)
      else map.set(index, next)
      return map
    })
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) {
          setStacksByIndex(new Map())
          setError(null)
        }
      }}
    >
      <DialogTrigger asChild>
        <Button
          type="button"
          size="sm"
          variant="default"
          className={cn('h-7 gap-1 text-xs', compact && 'shrink-0 px-2 sm:px-3')}
          disabled={disabled}
          aria-label={`Conjurar ${spell.name}`}
        >
          <Sparkles className="size-3.5" />
          <span className={compact ? 'hidden sm:inline' : undefined}>
            Conjurar
          </span>
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 font-display tracking-wide">
            <Zap className="size-5 text-[color:var(--primary)]" />
            {spell.name}
          </DialogTitle>
          <DialogDescription>
            Base {basePm} PM • Limite por magia {perSpellLimit} PM • PM
            atual {character.mpCurrent} / {character.mpMax}
          </DialogDescription>
        </DialogHeader>

        {spell.augments.length > 0 && spell.circle > 0 ? (
          <div className="space-y-2">
            <p
              className={cn(
                'text-[10px] uppercase tracking-widest',
                dimText,
              )}
            >
              Aprimoramentos
            </p>
            <ul className="space-y-2">
              {spell.augments.map((a, i) => {
                const stacks = stacksByIndex.get(i) ?? 0
                const lockedCircle =
                  a.requiresCircle !== undefined && a.requiresCircle > castableCircle
                return (
                  <li
                    key={i}
                    className={cn(
                      'flex flex-wrap items-start gap-2 rounded border border-border p-2',
                      lockedCircle && 'opacity-50',
                    )}
                  >
                    <div className="flex-1 space-y-0.5">
                      <p className="text-xs">
                        <span
                          className={cn(
                            'font-mono mr-2 text-[10px] uppercase tracking-widest',
                            a.kind === 'muda'
                              ? 'text-violet-700 dark:text-violet-300'
                              : 'text-emerald-700 dark:text-emerald-300',
                          )}
                        >
                          {a.kind}
                        </span>
                        +{a.pmCost} PM {a.kind === 'aumenta' ? 'cada' : ''}
                        {lockedCircle && (
                          <span className="ml-2 font-semibold text-red-700 dark:text-red-400">
                            requer {a.requiresCircle}º círculo
                          </span>
                        )}
                      </p>
                      <p className="text-xs text-foreground ">
                        {a.description}
                      </p>
                    </div>
                    {a.kind === 'muda' ? (
                      /* 'muda' é único por natureza — checkbox, não stepper. */
                      <input
                        type="checkbox"
                        checked={stacks > 0}
                        disabled={lockedCircle}
                        onChange={(e) => setStacks(i, e.target.checked ? 1 : 0)}
                        className="mt-1 size-5 accent-violet-600"
                        aria-label={`Aprimoramento: ${a.description.slice(0, 40)}`}
                      />
                    ) : (
                      <NumberInput
                        value={stacks}
                        onChange={(v) => setStacks(i, Math.max(0, v))}
                        min={0}
                        max={20}
                        disabled={lockedCircle}
                        className="w-20"
                        aria-label={`Aprimoramento ${i} — stacks`}
                      />
                    )}
                  </li>
                )
              })}
            </ul>
          </div>
        ) : (
          <p className={cn('text-xs italic', dimText)}>
            {spell.circle === 0
              ? 'Truques não aceitam aprimoramentos.'
              : 'Esta magia não possui aprimoramentos.'}
          </p>
        )}

        <div
          className={cn(
            'flex items-center justify-between rounded-lg border px-3 py-2',
            'border-border bg-muted  ',
          )}
        >
          <span
            className={cn(
              'text-xs uppercase tracking-widest',
              dimText,
            )}
          >
            Custo total
          </span>
          <span
            className={cn(
              'font-mono text-lg font-bold',
              castBlocked ? 'text-red-700 dark:text-red-400' : accentStrong,
            )}
          >
            {totalPm} PM
          </span>
        </div>

        {castBlocked && (
          <p className="text-xs text-red-700 dark:text-red-400">
            {castBlocked}
          </p>
        )}
        {error && <p className="text-xs text-destructive">{error}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button
            disabled={cast.isPending || Boolean(castBlocked)}
            onClick={() => cast.mutate()}
          >
            <Sparkles className="mr-1 size-4" />
            {cast.isPending ? 'Conjurando…' : 'Conjurar'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
