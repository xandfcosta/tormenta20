import { Link } from '@tanstack/react-router'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { api } from '@/lib/api'
import type { Character } from '@/lib/api'
import { invalidateCharacterDependents } from '@/lib/character-cache'
import { displacementTotal, useCharacterEffects } from '@/lib/derived'
import { characterQueryOptions } from '@/lib/queries'
import {
  accentBadge,
  accentTitle,
  dimText,
  subtleText,
  surface,
} from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
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
        'relative flex flex-wrap items-center justify-between gap-3 overflow-hidden rounded-xl px-4 py-3 sm:px-6',
        surface,
        'bg-gradient-to-r from-amber-200/70 via-amber-100 to-amber-200/70 dark:from-amber-900/40 dark:via-zinc-900 dark:to-amber-900/40',
        className,
      )}
    >
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_50%,rgba(217,119,6,0.18),transparent_50%)] dark:bg-[radial-gradient(circle_at_30%_50%,rgba(251,191,36,0.15),transparent_50%)]" />
      <div className="relative flex items-center gap-3">
        <Link to="/characters">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              subtleText,
              'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
            )}
          >
            ←
          </Button>
        </Link>
        <div className="min-w-0">
          <p className="text-[10px] uppercase tracking-[0.4em] text-amber-800/80 dark:text-amber-400/80">
            Tormenta 20
          </p>
          <h1
            className={cn(
              'truncate font-serif text-2xl font-bold tracking-tight sm:text-3xl',
              accentTitle,
            )}
          >
            {character.name}
          </h1>
          <p className={cn('mt-0.5 truncate text-xs', subtleText)}>
            {races.join(' / ')} • {character.origin}
            {character.god && (
              <>
                {' '}
                •{' '}
                <span className="text-amber-700 dark:text-amber-300">
                  {character.god}
                </span>
              </>
            )}
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
      <div className="relative flex items-center gap-2">
        <div className="flex flex-wrap justify-end gap-1">
          {character.classes.map((c) => (
            <Badge key={c.className} className={accentBadge}>
              {c.className} {c.level}
            </Badge>
          ))}
        </div>
        <LevelBadge character={character} />
        <Link
          to="/characters/$id/sheet"
          params={{ id: String(character.id) }}
        >
          <Button variant="outline" size="sm">
            Ficha computada
          </Button>
        </Link>
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
    if (next < 1 || next > 20) return
    if (character.level + delta < 1 || character.level + delta > 20) return
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
      <div
        className={cn(
          'flex items-center gap-1 rounded-lg border px-2 py-1 text-center',
          'border-amber-700/40 bg-amber-50/80 dark:border-amber-500/40 dark:bg-zinc-950/80',
        )}
      >
        <button
          type="button"
          onClick={() => trigger('down')}
          disabled={atMin || mutate.isPending}
          aria-label="Diminuir nível"
          className={cn(
            'text-amber-700 transition-colors disabled:opacity-30 dark:text-amber-300',
            'hover:text-amber-900 dark:hover:text-amber-100',
          )}
        >
          <ChevronDown className="size-4" />
        </button>
        <div className="flex flex-col items-center leading-none">
          <p className={cn('text-[9px] uppercase tracking-widest', subtleText)}>
            Nv
          </p>
          <p
            className={cn(
              'w-7 text-center font-serif text-2xl font-bold leading-none text-amber-700 dark:text-amber-300',
            )}
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
          className={cn(
            'text-amber-700 transition-colors disabled:opacity-30 dark:text-amber-300',
            'hover:text-amber-900 dark:hover:text-amber-100',
          )}
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
          <p className={cn('text-xs italic', dimText)}>
            Nenhuma classe elegível.
          </p>
        ) : (
          <ul className="space-y-1.5">
            {eligible.map((c) => (
              <li key={c.className}>
                <button
                  type="button"
                  onClick={() => onPick(c.className)}
                  className={cn(
                    'flex w-full items-center justify-between rounded border px-3 py-2 text-left transition-colors',
                    'border-amber-700/30 hover:bg-amber-100 dark:border-amber-500/30 dark:hover:bg-amber-500/10',
                  )}
                >
                  <span className={cn('text-sm font-semibold', accentTitle)}>
                    {c.className}
                  </span>
                  <span className={cn('text-xs', subtleText)}>
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
          className="cursor-help font-semibold text-amber-700 underline decoration-dotted underline-offset-2 dark:text-amber-300"
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
