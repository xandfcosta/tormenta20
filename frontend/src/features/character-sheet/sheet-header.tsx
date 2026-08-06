import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { isValid, validateTotalLevel } from '@tormenta20/t20-data'
import { ChevronDown, ChevronUp } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/shared/ui/tooltip'
import { api } from '@/shared/api/api'
import type { Character, ClassLevelResult } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import {
  characterEffects,
  useCharacterEffects,
} from '@/entities/character/derived'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import type { ValueBreakdown } from '@/shared/lib/computed-sheet-v2'
import { optimisticLevelVitals } from '@/entities/character/level-vitals'
import { characterQueryOptions } from '@/entities/character/queries'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

/**
 * Character name + one-line subtitle (races · origin · devoção · size ·
 * displacement, plus a fatigue warning when relevant). The text half of the
 * HUD identity block; the portrait, class badges and level stepper are
 * composed alongside it by `CharacterHud`.
 */
export function SheetIdentityText({ character }: { character: Character }) {
  const races = character.races.map((r) => r.race)
  const sheet = useComputedSheet(character)
  const disp = sheet.displacement
  const fly = sheet.flySpeed
  // The fatigue-on-sleep flag isn't a breakdown — read it from the raw effects
  // (heavy-armor rest penalty); everything numeric comes from the sheet.
  const fatigue = useCharacterEffects(character).flags.has('fatigue-on-sleep')
  return (
    <div className="min-w-0">
      <h1 className="truncate text-lg font-bold leading-tight tracking-tight sm:text-xl">
        {character.name}
      </h1>
      <p className="line-clamp-1 text-xs leading-tight text-muted-foreground sm:line-clamp-2">
        {races.join(' / ')} • {character.origin}
        {' • '}
        <span className="text-foreground">
          {character.god ?? 'Sem devoção'}
          {character.god && character.godPower
            ? ` (${character.godPower})`
            : ''}
        </span>
        {' • '}
        {character.size} • <DisplacementBadge disp={disp} />
        {fly > 0 && (
          <>
            {' • '}
            <span className="text-foreground">voo {fly}m</span>
          </>
        )}
        {fatigue && (
          <>
            {' • '}
            <FatigueWarning />
          </>
        )}
      </p>
    </div>
  )
}

export function LevelBadge({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const [pickerDir, setPickerDir] = useState<null | 'up' | 'down'>(null)

  const mutate = useMutation<
    ClassLevelResult,
    Error,
    { className: string; level: number },
    { previous: Character | undefined }
  >({
    mutationFn: (input) =>
      api.characters.updateClassLevel(character.id, input),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        const nextClasses = prev.classes.map((c) =>
          c.className === input.className ? { ...c, level: input.level } : c,
        )
        const total = nextClasses.reduce((s, c) => s + c.level, 0)
        // Optimistic PV/PM: same shared engine pools + current-shift rule the
        // server applies, so the bars move with the stepper instead of waiting
        // a roundtrip. onSuccess reconciles with the authoritative delta.
        const vitals = optimisticLevelVitals(
          prev,
          characterEffects(prev),
          nextClasses,
        )
        return { ...prev, classes: nextClasses, level: total, ...vitals }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (delta) => {
      // `vitals` rides along because PV/PM max are DERIVED from class levels —
      // without merging them the bars kept the pre-level pools (2026-08 bug).
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev
          ? { ...prev, level: delta.level, classes: delta.classes, ...delta.vitals }
          : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const bumpClass = (className: string, delta: 1 | -1) => {
    const entry = character.classes.find((c) => c.className === className)
    if (!entry) return
    const next = entry.level + delta
    // Shared rule: every class level ≥ 1 and the total ≤ 20 (same guard the
    // backend applies), so the optimistic level bump can't be rejected.
    const projected = character.classes.map((c) =>
      c.className === className ? { ...c, level: next } : c,
    )
    if (!isValid(validateTotalLevel(projected))) return
    mutate.mutate({ className, level: next })
  }

  const trigger = (dir: 'up' | 'down') => {
    if (character.classes.length === 0) return
    if (character.classes.length === 1) {
      bumpClass(character.classes[0].className, dir === 'up' ? 1 : -1)
      return
    }
    setPickerDir(dir)
  }

  const atMin = character.level <= 1
  const atMax = character.level >= 20

  return (
    <>
      <div className="flex items-center gap-0.5 rounded-lg border bg-muted px-1 py-0.5 text-center sm:gap-1 sm:px-2 sm:py-1">
        <button
          type="button"
          onClick={() => trigger('down')}
          disabled={atMin || mutate.isPending}
          aria-label="Diminuir nível"
          className="flex size-7 items-center justify-center text-foreground transition-colors hover:text-foreground disabled:opacity-30 sm:size-6"
        >
          <ChevronDown className="size-3.5 sm:size-4" />
        </button>
        <div className="flex flex-col items-center leading-none">
          <p className="text-[8px] uppercase tracking-widest text-muted-foreground sm:text-[9px]">
            Nv
          </p>
          <p
            className="w-5 text-center text-lg font-bold leading-none text-foreground sm:w-7 sm:text-2xl"
            aria-label="Nível"
          >
            {character.level}
          </p>
        </div>
        <button
          type="button"
          onClick={() => trigger('up')}
          disabled={atMax || mutate.isPending}
          aria-label="Aumentar nível"
          className="flex size-7 items-center justify-center text-foreground transition-colors hover:text-foreground disabled:opacity-30 sm:size-6"
        >
          <ChevronUp className="size-3.5 sm:size-4" />
        </button>
      </div>
      {pickerDir && (
        <ClassLevelPicker
          character={character}
          direction={pickerDir}
          onPick={(className) => {
            const delta = pickerDir === 'up' ? 1 : -1
            bumpClass(className, delta)
            setPickerDir(null)
          }}
          onClose={() => setPickerDir(null)}
        />
      )}
    </>
  )
}

function ClassLevelPicker({
  character,
  direction,
  onPick,
  onClose,
}: {
  character: Character
  direction: 'up' | 'down'
  onPick: (className: string) => void
  onClose: () => void
}) {
  const eligible = character.classes.filter((c) =>
    direction === 'up' ? c.level < 20 : c.level > 1,
  )
  const verb = direction === 'up' ? 'Subir' : 'Reduzir'
  return (
    <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{verb} nível — escolha a classe</DialogTitle>
        </DialogHeader>
        {eligible.length === 0 ? (
          <p className="text-xs italic text-muted-foreground">
            Nenhuma classe elegível.
          </p>
        ) : (
          <ul className="space-y-1.5">
            {eligible.map((c) => (
              <li key={c.className}>
                <button
                  type="button"
                  onClick={() => onPick(c.className)}
                  className="flex w-full items-center justify-between rounded border px-3 py-2 text-left transition-colors hover:bg-muted"
                >
                  <span className="text-sm font-semibold text-foreground">
                    {c.className}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {c.level} → {direction === 'up' ? c.level + 1 : c.level - 1}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </DialogContent>
    </Dialog>
  )
}

function DisplacementBadge({
  disp,
}: {
  disp: ValueBreakdown
}) {
  const changed = disp.itemBonus !== 0
  if (!changed) return <span>{disp.total}m</span>
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className={cn(
            'cursor-help underline decoration-dotted underline-offset-2',
            disp.itemBonus < 0
              ? 'text-red-700 dark:text-red-300'
              : 'text-emerald-700 dark:text-emerald-300',
          )}
        >
          {disp.total}m
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="text-xs">
          <div>Base {disp.base}m</div>
          {disp.contributions.map((c, i) => (
            <div key={i}>
              {c.source} {signed(c.amount)}m
            </div>
          ))}
        </div>
      </TooltipContent>
    </Tooltip>
  )
}

function FatigueWarning() {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          className="cursor-help font-semibold text-foreground underline decoration-dotted underline-offset-2 "
        >
          Fadiga ao dormir
        </button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <div className="max-w-[260px] text-xs">
          Dormir vestindo armadura pesada causa Fadiga (1 condição). Remova a
          armadura antes de descansar.
        </div>
      </TooltipContent>
    </Tooltip>
  )
}
