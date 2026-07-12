import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Dumbbell, Star, Trash2 } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/shared/ui/tooltip'
import type {
  AttributeKey,
  Character,
  CharacterExpertise,
} from '@/shared/api/api'
import { api } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { expertiseTotalWithItems } from '@/entities/character/derived'
import type { ExpertiseDef } from '@/entities/character/expertise'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  expertiseStateFor,
  trainingBonusForLevel,
} from '@/entities/character/expertise'
import { characterQueryOptions } from '@/entities/character/queries'
import {
  accentStrong,
  hoverRow,
  selectClass,
  subtleText,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
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
    <TrainedToggle
      trained={state.trained}
      name={def.name}
      onToggle={(next) => mutation.mutate({ trained: next })}
    />
  )

  const locked = !!def.trainedOnly && !state.trained

  const attrSelect = (
    <select
      value={state.attribute}
      onChange={(e) =>
        mutation.mutate({ attribute: e.target.value as AttributeKey })
      }
      className={cn(selectClass, 'h-6 rounded-full px-2 font-mono text-[11px]')}
      aria-label={`${def.name} atributo`}
    >
      {ATTRIBUTE_KEYS.map((k) => (
        <option key={k} value={k}>
          {ATTRIBUTE_ABBR[k]} {signed(character[k])}
        </option>
      ))}
    </select>
  )

  return (
    <ExpertiseBreakdown
      name={def.name}
      total={total}
      locked={locked}
      halfLevel={halfLevel}
      attrAbbr={ATTRIBUTE_ABBR[state.attribute]}
      attrMod={character[state.attribute]}
      trainBonus={trainBonus}
      itemBonus={othersDisplay}
      contributions={detail.itemContributions}
    >
      <div
        className={cn(
          'flex items-start gap-2.5 rounded-lg border p-2.5',
          state.trained
            ? 'border-amber-500/50 bg-amber-500/[0.06]'
            : 'border-amber-700/15 dark:border-amber-500/10',
          hoverRow,
        )}
      >
        {/* Both the badge and the name open the breakdown; the toggle, attr
            select and delete stay interactive (they are not triggers). */}
        <DialogTrigger asChild>
          <TotalBadge total={total} locked={locked} />
        </DialogTrigger>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <DialogTrigger asChild>
              <button
                type="button"
                className={cn(
                  'min-w-0 flex-1 truncate text-left text-sm hover:underline',
                  locked
                    ? 'text-zinc-500 dark:text-zinc-500'
                    : 'text-zinc-800 dark:text-zinc-200',
                )}
              >
                {def.name}
              </button>
            </DialogTrigger>
            {def.trainedOnly && <TrainedOnlyStar locked={locked} />}
            {trainedToggle}
            {onDelete && (
              <DeleteExpertiseButton name={def.name} onDelete={onDelete} />
            )}
          </div>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            {attrSelect}
            <Chip label="½lvl" value={String(halfLevel)} />
            <Chip label="treino" value={signed(trainBonus)} />
            <DialogTrigger asChild>
              <button type="button" className="inline-flex hover:brightness-105">
                <Chip label="outros" value={signed(othersDisplay)} />
              </button>
            </DialogTrigger>
          </div>
        </div>
      </div>
    </ExpertiseBreakdown>
  )
}

/** Star marking a trained-only perícia; amber once it's locked (untrained). */
function TrainedOnlyStar({ locked }: { locked: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
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
  )
}

/** Prominent skill total, doubling as the trigger that opens the modifier
 *  breakdown. Amber when usable, struck-through when trained-only + untrained. */
function TotalBadge({ total, locked }: { total: number; locked: boolean }) {
  return (
    <button
      type="button"
      aria-label="Ver detalhamento dos modificadores"
      className={cn(
        'flex size-11 shrink-0 items-center justify-center rounded-lg border font-mono text-lg font-bold transition-colors hover:brightness-110',
        locked
          ? 'border-zinc-300 text-zinc-400 line-through dark:border-zinc-700 dark:text-zinc-600'
          : ['border-amber-500/40 bg-amber-500/10', accentStrong],
      )}
    >
      {signed(total)}
    </button>
  )
}

/** Small self-labeling breakdown chip (½lvl / treino). */
function Chip({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-amber-700/20 bg-zinc-100/50 px-2 py-0.5 font-mono text-[11px] text-zinc-700 dark:border-amber-500/15 dark:bg-zinc-900/40 dark:text-zinc-300">
      <span className="text-[9px] uppercase tracking-wider opacity-60">
        {label}
      </span>
      {value}
    </span>
  )
}

/**
 * Trained toggle — replaces the raw browser checkbox with a themed switch that
 * reads on the dark/amber sheet: a dumbbell that fills amber when trained.
 * `role="switch"` keeps it a first-class control for keyboard + screen readers.
 */
function TrainedToggle({
  trained,
  name,
  onToggle,
}: {
  trained: boolean
  name: string
  onToggle: (next: boolean) => void
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          role="switch"
          aria-checked={trained}
          aria-label={`${name} treinada`}
          onClick={() => onToggle(!trained)}
          className={cn(
            'inline-flex size-6 shrink-0 items-center justify-center rounded-md border transition-colors',
            trained
              ? 'border-amber-500 bg-amber-500 text-zinc-950 shadow-sm dark:border-amber-400 dark:bg-amber-400'
              : 'border-amber-700/30 text-amber-700/40 hover:border-amber-600 hover:text-amber-600 dark:border-amber-500/25 dark:text-amber-500/30',
          )}
        >
          <Dumbbell className="size-3.5" strokeWidth={2.5} />
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">
        {trained ? 'Treinada' : 'Não treinada'}
      </TooltipContent>
    </Tooltip>
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

type ItemContributions =
  ReturnType<typeof expertiseTotalWithItems>['itemContributions']

/**
 * Uniform modifier breakdown for a perícia — the same for every skill, not
 * just those with item bonuses: ½ nível + atributo + treino + outros (with
 * per-item lines) summing to the total. Opened from the total badge.
 */
function ExpertiseBreakdown({
  name,
  total,
  locked,
  halfLevel,
  attrAbbr,
  attrMod,
  trainBonus,
  itemBonus,
  contributions,
  children,
}: {
  name: string
  total: number
  locked: boolean
  halfLevel: number
  attrAbbr: string
  attrMod: number
  trainBonus: number
  itemBonus: number
  contributions: ItemContributions
  children: React.ReactNode
}) {
  return (
    <Dialog>
      {children}
      <DialogContent
        className={cn(
          'w-[calc(100vw-1.5rem)] max-w-[calc(100vw-1.5rem)] p-4 sm:w-full sm:max-w-sm sm:p-6',
          'border-amber-700/40 bg-amber-50 text-zinc-900 dark:border-amber-500/40 dark:bg-zinc-950 dark:text-zinc-100',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn('font-serif', accentStrong)}>
            {name}
          </DialogTitle>
        </DialogHeader>
        <div className="space-y-1 text-sm">
          <BreakdownRow label="½ nível" value={halfLevel} />
          <BreakdownRow label={`Atributo (${attrAbbr})`} value={attrMod} />
          <BreakdownRow label="Treino" value={trainBonus} />
          <BreakdownRow label="Outros" value={itemBonus} />
          {contributions.map((c) => (
            <BreakdownRow
              key={`${c.source}-${c.amount}`}
              label={c.source}
              value={c.amount}
              indented
            />
          ))}
          <div
            className={cn(
              'mt-2 flex items-center justify-between rounded-lg border px-3 py-2',
              'border-amber-700/40 bg-amber-100/60 dark:border-amber-500/40 dark:bg-amber-950/30',
            )}
          >
            <span
              className={cn('text-xs uppercase tracking-widest', subtleText)}
            >
              Total
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
        </div>
      </DialogContent>
    </Dialog>
  )
}

function BreakdownRow({
  label,
  value,
  indented,
}: {
  label: string
  value: number
  indented?: boolean
}) {
  return (
    <div
      className={cn(
        'flex items-center justify-between gap-2 border-b border-amber-700/10 py-1 dark:border-amber-500/10',
        indented && 'pl-4 text-xs opacity-80',
      )}
    >
      <span className="truncate">{label}</span>
      <span className="shrink-0 font-mono">{signed(value)}</span>
    </div>
  )
}

