import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { isValid, validateTotalLevel } from '@tormenta20/t20-data'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
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
import type { Character } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { displacementTotal, useCharacterEffects } from '@/entities/character/derived'
import { characterQueryOptions } from '@/entities/character/queries'
import { cn } from '@/shared/lib/utils'
import { signed } from './signed'

export function SheetHeader({
  character,
  className,
}: {
  character: Character
  className?: string
}) {
  const races = character.races.map((r) => r.race)
  const effects = useCharacterEffects(character)
  const disp = displacementTotal(character, effects)
  const fatigue = effects.flags.has('fatigue-on-sleep')
  return (
    <header
      className={cn(
        'flex flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-xl border bg-card px-4 py-2.5',
        className,
      )}
    >
      <div className="flex min-w-0 items-center gap-3">
        <CharacterPortrait name={character.name} size="sm" />
        <div className="min-w-0">
          <h1 className="truncate text-xl font-bold leading-tight tracking-tight sm:text-2xl">
            {character.name}
          </h1>
          <p className="truncate text-xs text-muted-foreground">
            {races.join(' / ')} • {character.origin}
            {' • '}
            <span className="text-foreground">
              {character.god ?? 'Sem devoção'}
            </span>
            {' • '}
            {character.size} • <DisplacementBadge disp={disp} />
            {fatigue && (
              <>
                {' • '}
                <FatigueWarning />
              </>
            )}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <div className="flex flex-wrap justify-end gap-1">
          {character.classes.map((c) => (
            <Badge key={c.className}>
              {c.className} {c.level}
            </Badge>
          ))}
        </div>
        <LevelBadge character={character} />
      </div>
    </header>
  )
}

function LevelBadge({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const [pickerDir, setPickerDir] = useState<null | 'up' | 'down'>(null)

  const mutate = useMutation<
    Character,
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
        return { ...prev, classes: nextClasses, level: total }
      })
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => {
      qc.setQueryData<Character>(queryKey, server)
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
      <div className="flex items-center gap-1 rounded-lg border bg-muted px-2 py-1 text-center">
        <button
          type="button"
          onClick={() => trigger('down')}
          disabled={atMin || mutate.isPending}
          aria-label="Diminuir nível"
          className="text-foreground transition-colors hover:text-foreground disabled:opacity-30"
        >
          <ChevronDown className="size-4" />
        </button>
        <div className="flex flex-col items-center leading-none">
          <p className="text-[9px] uppercase tracking-widest text-muted-foreground">
            Nv
          </p>
          <p
            className="w-7 text-center text-2xl font-bold leading-none text-foreground"
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
          className="text-foreground transition-colors hover:text-foreground disabled:opacity-30"
        >
          <ChevronUp className="size-4" />
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
  disp: ReturnType<typeof displacementTotal>
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
