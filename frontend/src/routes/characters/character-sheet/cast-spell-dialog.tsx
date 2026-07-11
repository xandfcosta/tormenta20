import { useMemo, useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Sparkles, Zap } from 'lucide-react'
import type { CatalogSpell } from '@tormenta20/t20-data'
import { SPELL_BASE_PM_COST } from '@tormenta20/t20-data'
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
import type { Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/shared/lib/character-cache'
import { characterQueryOptions } from '@/shared/lib/queries'
import { accentStrong, dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'

type AugmentPick = { augmentIndex: number; stacks: number }

/**
 * Cast dialog — user picks stacks per augment (0 = not taken; `muda`
 * augments capped at 1). Shows a live PM total + per-spell limit
 * check. Server is authoritative — the client-side preview is only a
 * UX hint.
 */
export function CastSpellDialog({
  spell,
  character,
  disabled,
}: {
  spell: CatalogSpell
  character: Character
  disabled?: boolean
}) {
  const [open, setOpen] = useState(false)
  const [stacksByIndex, setStacksByIndex] = useState<Map<number, number>>(
    new Map(),
  )
  const [error, setError] = useState<string | null>(null)

  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey

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
  const perSpellLimit = Math.max(1, Math.floor(character.level / 2))
  const overLimit = spell.circle > 0 && totalPm > perSpellLimit
  const insufficientPm = totalPm > character.mpCurrent

  const cast = useMutation({
    mutationFn: () =>
      api.characters.castSpell(character.id, spell.id, augmentPicks),
    onSuccess: (updated) => {
      qc.setQueryData(queryKey, updated)
      invalidateCharacterDependents(qc, character.id)
      setOpen(false)
      setStacksByIndex(new Map())
      setError(null)
    },
    onError: (e) => {
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
          className="h-7 gap-1 text-xs"
          disabled={disabled}
        >
          <Sparkles className="size-3.5" />
          Conjurar
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
                return (
                  <li
                    key={i}
                    className="flex flex-wrap items-start gap-2 rounded border border-amber-700/20 p-2 dark:border-amber-500/15"
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
                      </p>
                      <p className="text-xs text-zinc-700 dark:text-zinc-300">
                        {a.description}
                      </p>
                    </div>
                    <NumberInput
                      value={stacks}
                      onChange={(v) => setStacks(i, Math.max(0, v))}
                      min={0}
                      max={a.kind === 'muda' ? 1 : 20}
                      className="w-20"
                      aria-label={`Aprimoramento ${i} — stacks`}
                    />
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
            'border-amber-700/40 bg-amber-100/60 dark:border-amber-500/40 dark:bg-zinc-900/60',
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
              overLimit || insufficientPm
                ? 'text-red-700 dark:text-red-400'
                : accentStrong,
            )}
          >
            {totalPm} PM
          </span>
        </div>

        {overLimit && (
          <p className="text-xs text-red-700 dark:text-red-400">
            Custo total excede o limite por magia ({perSpellLimit} PM).
          </p>
        )}
        {insufficientPm && !overLimit && (
          <p className="text-xs text-red-700 dark:text-red-400">
            PM insuficientes ({character.mpCurrent} disponíveis).
          </p>
        )}
        {error && <p className="text-xs text-destructive">{error}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            Cancelar
          </Button>
          <Button
            disabled={cast.isPending || overLimit || insufficientPm}
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
