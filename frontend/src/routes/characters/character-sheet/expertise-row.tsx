import { useMutation, useQueryClient } from '@tanstack/react-query'
import { SlidersHorizontal, Star, Trash2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
} from '@/lib/api'
import { api } from '@/lib/api'
import { invalidateCharacterDependents } from '@/lib/character-cache'
import { expertiseTotalWithItems } from '@/lib/derived'
import type { ExpertiseDef } from '@/lib/expertise'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  expertiseStateFor,
  trainingBonusForLevel,
} from '@/lib/expertise'
import { characterQueryOptions } from '@/lib/queries'
import {
  accentStrong,
  dimText,
  hoverRow,
  selectClass,
  subtleText,
} from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
import type { ItemEffects } from '@tormenta20/t20-data'
import { signed } from './signed'

export function ExpertiseRow({
  character,
  def,
  effects,
  onDelete,
}: {
  character: Character
  def: ExpertiseDef
  effects: ItemEffects
  onDelete?: () => void
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const state = expertiseStateFor(character, def)

  type ExpertisePatch = {
    attribute?: AttributeKey
    trained?: boolean
  }

  const mutation = useMutation<
    CharacterExpertise,
    Error,
    ExpertisePatch,
    { previous: Character | undefined }
  >({
    mutationFn: (input) =>
      api.characters.updateExpertise(character.id, { name: def.name, ...input }),
    onMutate: async (input) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: prev.expertises.map((e) =>
            e.name === def.name ? { ...e, ...input } : e,
          ),
        }
      })
      return { previous }
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (updated) => {
      qc.setQueryData<Character>(queryKey, (prev) => {
        if (!prev) return prev
        return {
          ...prev,
          expertises: prev.expertises.map((e) =>
            e.name === updated.name ? updated : e,
          ),
        }
      })
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const detail = expertiseTotalWithItems(character, state, effects)
  const total = detail.total
  const halfLevel = Math.floor(character.level / 2)
  const trainBonus = state.trained ? trainingBonusForLevel(character.level) : 0
  const othersDisplay = detail.itemBonus

  const trainedToggle = (
    <input
      type="checkbox"
      className="h-4 w-4 cursor-pointer accent-amber-600 dark:accent-amber-500"
      checked={state.trained}
      onChange={(e) => mutation.mutate({ trained: e.target.checked })}
      aria-label={`${def.name} treinada`}
    />
  )

  const locked = !!def.trainedOnly && !state.trained

  const totalLabel = (
    <span
      className={cn(
        'font-mono text-base font-semibold',
        locked
          ? 'text-zinc-400 line-through dark:text-zinc-600'
          : accentStrong,
      )}
      title={locked ? 'Apenas treinada — não pode ser usada sem treino' : undefined}
    >
      {signed(total)}
    </span>
  )

  const nameNode = (
    <span
      className={cn(
        'flex flex-1 items-center gap-1.5 truncate text-sm',
        locked
          ? 'text-zinc-500 dark:text-zinc-500'
          : 'text-zinc-800 dark:text-zinc-200',
      )}
    >
      <span className="truncate">{def.name}</span>
      {def.trainedOnly && (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              tabIndex={0}
              aria-label="Apenas treinada"
              className="inline-flex shrink-0 cursor-help"
            >
              <Star
                className={cn(
                  'size-3',
                  locked
                    ? 'fill-amber-500 text-amber-700 dark:fill-amber-400 dark:text-amber-300'
                    : 'fill-zinc-300 text-zinc-500 dark:fill-zinc-700 dark:text-zinc-500',
                )}
              />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">
            Pode ser usada apenas quando treinada
          </TooltipContent>
        </Tooltip>
      )}
    </span>
  )

  const othersInput = (
    <OthersDisplay
      total={othersDisplay}
      detail={detail}
      expertiseName={def.name}
    />
  )

  return (
    <>
      {/* Mobile: compact row + dialog */}
      <div
        className={cn(
          'flex items-center gap-2 rounded-md px-2 py-1.5 sm:hidden',
          hoverRow,
        )}
      >
        {trainedToggle}
        {nameNode}
        <span className="w-10 text-right">{totalLabel}</span>
        {onDelete && <DeleteExpertiseButton name={def.name} onDelete={onDelete} />}
        <Dialog>
          <DialogTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className={cn(
                'size-7',
                subtleText,
                'hover:bg-amber-200/60 hover:text-amber-900 dark:hover:bg-zinc-800/60 dark:hover:text-amber-200',
              )}
              aria-label={`Editar ${def.name}`}
            >
              <SlidersHorizontal className="size-4" />
            </Button>
          </DialogTrigger>
          <DialogContent
            className={cn(
              'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
              'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
            )}
          >
            <DialogHeader>
              <DialogTitle
                className={cn('flex items-center gap-2 font-serif', accentStrong)}
              >
                {def.name}
                {def.trainedOnly && (
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <button
                        type="button"
                        aria-label="Apenas treinada"
                        className="inline-flex cursor-help"
                      >
                        <Star
                          className={cn(
                            'size-4',
                            locked
                              ? 'fill-amber-500 text-amber-700 dark:fill-amber-400 dark:text-amber-300'
                              : 'fill-zinc-300 text-zinc-500 dark:fill-zinc-700 dark:text-zinc-500',
                          )}
                        />
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      Pode ser usada apenas quando treinada
                    </TooltipContent>
                  </Tooltip>
                )}
              </DialogTitle>
            </DialogHeader>
            {def.trainedOnly && !state.trained && (
              <p
                className={cn(
                  'rounded-md border px-3 py-2 text-xs',
                  'border-amber-700/40 bg-amber-100/60 text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
                )}
              >
                Esta perícia exige treino para ser usada.
              </p>
            )}
            <div className="space-y-4">
              <div
                className={cn(
                  'flex items-center justify-between rounded-lg border px-4 py-2',
                  'border-amber-700/30 bg-amber-100/70 dark:border-amber-500/30 dark:bg-zinc-900/60',
                )}
              >
                <span
                  className={cn(
                    'text-xs uppercase tracking-widest',
                    subtleText,
                  )}
                >
                  total
                </span>
                <span
                  className={cn(
                    'font-mono text-2xl font-bold',
                    locked
                      ? 'text-zinc-400 line-through dark:text-zinc-500'
                      : accentStrong,
                  )}
                >
                  {signed(total)}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <DialogField label="½ nível">
                  <span className="font-mono text-sm text-zinc-800 dark:text-zinc-200">
                    {halfLevel}
                  </span>
                </DialogField>
                <DialogField label="atributo">
                  <select
                    value={state.attribute}
                    onChange={(e) =>
                      mutation.mutate({
                        attribute: e.target.value as AttributeKey,
                      })
                    }
                    className={cn(selectClass, 'h-7 px-2 font-mono text-xs')}
                    aria-label={`${def.name} atributo`}
                  >
                    {ATTRIBUTE_KEYS.map((k) => (
                      <option key={k} value={k}>
                        {ATTRIBUTE_ABBR[k]} {signed(character[k])}
                      </option>
                    ))}
                  </select>
                </DialogField>
                <DialogField label="treino">
                  <div className="flex items-center gap-2">
                    {trainedToggle}
                    <span className="font-mono text-sm text-zinc-800 dark:text-zinc-200">
                      {signed(trainBonus)}
                    </span>
                  </div>
                </DialogField>
                <DialogField label="outros">{othersInput}</DialogField>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Desktop: full breakdown row */}
      <div
        className={cn(
          'hidden items-center gap-2 rounded-md px-2 py-1 sm:flex',
          hoverRow,
        )}
      >
        {trainedToggle}
        {nameNode}
        <span className="w-10 text-right">{totalLabel}</span>
        <span className="w-8 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {halfLevel}
        </span>
        <select
          value={state.attribute}
          onChange={(e) =>
            mutation.mutate({ attribute: e.target.value as AttributeKey })
          }
          className={cn(selectClass, 'h-7 w-16 px-1 font-mono text-[11px]')}
          aria-label={`${def.name} atributo`}
        >
          {ATTRIBUTE_KEYS.map((k) => (
            <option key={k} value={k}>
              {ATTRIBUTE_ABBR[k]} {signed(character[k])}
            </option>
          ))}
        </select>
        <span className="w-10 text-center font-mono text-xs text-zinc-700 dark:text-zinc-300">
          {signed(trainBonus)}
        </span>
        <div className="w-20">
          <OthersDisplay
            total={othersDisplay}
            detail={detail}
            expertiseName={def.name}
          />
        </div>
        {onDelete ? (
          <DeleteExpertiseButton name={def.name} onDelete={onDelete} />
        ) : (
          <span className="size-7 shrink-0" aria-hidden />
        )}
      </div>
    </>
  )
}

function DeleteExpertiseButton({
  name,
  onDelete,
}: {
  name: string
  onDelete: () => void
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-zinc-500 hover:bg-red-100 hover:text-red-700 dark:text-zinc-400 dark:hover:bg-red-950/40 dark:hover:text-red-400"
          onClick={() => {
            if (confirm(`Remover ofício "${name}"?`)) onDelete()
          }}
          aria-label={`Remover ${name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>Remover ofício</TooltipContent>
    </Tooltip>
  )
}

function OthersDisplay({
  total,
  detail,
  expertiseName,
}: {
  total: number
  detail: ReturnType<typeof expertiseTotalWithItems>
  expertiseName: string
}) {
  const hasContribs = detail.itemContributions.length > 0
  const display = (
    <span
      className={cn(
        'block w-full rounded-md border bg-zinc-100/60 px-2 py-1 text-center font-mono text-xs',
        'border-amber-700/20 text-zinc-700 dark:border-amber-500/20 dark:bg-zinc-900/40 dark:text-zinc-300',
        total === 0 && 'text-zinc-400 dark:text-zinc-600',
        hasContribs &&
          'cursor-pointer hover:bg-amber-100/60 dark:hover:bg-zinc-800/60',
      )}
    >
      {signed(total)}
    </span>
  )
  if (!hasContribs) return display
  return (
    <Dialog>
      <DialogTrigger asChild>
        <button
          type="button"
          className="block w-full"
          aria-label={`Detalhes de Outros — ${expertiseName}`}
        >
          {display}
        </button>
      </DialogTrigger>
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle
            className={cn('flex items-center gap-2 font-serif', accentStrong)}
          >
            Outros — {expertiseName}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-2 text-sm">
          <ul className="space-y-1">
            {detail.itemContributions.map((c, i) => (
              <li
                key={i}
                className="flex items-center justify-between gap-2 border-b border-amber-700/15 pb-1 dark:border-amber-500/10"
              >
                <span className="truncate">{c.source}</span>
                <span className="shrink-0 font-mono">{signed(c.amount)}</span>
              </li>
            ))}
          </ul>
          <div
            className={cn(
              'flex items-center justify-between rounded-lg border px-3 py-2',
              'border-amber-700/40 bg-amber-100/60 dark:border-amber-500/40 dark:bg-amber-950/30',
            )}
          >
            <span
              className={cn(
                'text-xs uppercase tracking-widest',
                subtleText,
              )}
            >
              Total
            </span>
            <span
              className={cn(
                'font-mono text-2xl font-bold',
                accentStrong,
              )}
            >
              {signed(total)}
            </span>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function DialogField({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-1">
      <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
        {label}
      </span>
      {children}
    </div>
  )
}
