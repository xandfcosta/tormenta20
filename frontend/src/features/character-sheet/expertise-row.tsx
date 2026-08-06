import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Dumbbell, Lock, Star, Trash2 } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { ConfirmDialog } from '@/shared/ui/confirm-dialog'
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
import { expertiseFromSheet } from '@/entities/character/computed-sheet'
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
import type {
  BreakdownContribution,
  ComputedSheetV2,
} from '@/shared/lib/computed-sheet-v2'
import { signed } from './signed'

export function ExpertiseRow({
  character,
  def,
  sheet,
  onDelete,
}: {
  character: Character
  def: ExpertiseDef
  sheet: ComputedSheetV2
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

  // Every standard + custom perícia is on the sheet; `?? 0` is only a type guard.
  const entry = expertiseFromSheet(sheet, def.name)
  const total = entry?.total ?? 0
  const attrMod = entry?.attrValue ?? 0
  const halfLevel = Math.floor(character.level / 2)
  const trainBonus = state.trained ? trainingBonusForLevel(character.level) : 0
  const othersDisplay = entry?.itemBonus ?? 0
  const itemContributions = entry?.itemContributions ?? []

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
      {/* Final modifier (race/item bonuses in), not the raw sheet value —
          the row must agree with the breakdown + total (bug C). */}
      {ATTRIBUTE_KEYS.map((k) => (
        <option key={k} value={k}>
          {ATTRIBUTE_ABBR[k]} {signed(sheet.attributes[k].total)}
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
      attrMod={attrMod}
      trainBonus={trainBonus}
      itemBonus={othersDisplay}
      contributions={itemContributions}
    >
      <div
        className={cn(
          'flex items-start gap-2.5 rounded-lg border p-2.5',
          state.trained
            ? 'border-border bg-muted/[0.06]'
            : 'border-border ',
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
                    ? 'text-foreground '
                    : 'text-foreground ',
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
              // Amber = locked out (trained-only, still untrained); subtle
              // outline once trained so the marker stops shouting.
              locked
                ? 'fill-amber-500 text-amber-500'
                : 'fill-none text-muted-foreground/60',
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
 *  breakdown. Locked (trained-only + untrained) = dashed border, dimmed value
 *  and an amber corner padlock — the old line-through was illegible on the
 *  small mono digits (2026-08 owner report). Rest props are spread onto the
 *  button: `DialogTrigger asChild` injects its onClick/ref via props, and
 *  dropping them left the trigger dead (bug D). */
function TotalBadge({
  total,
  locked,
  className,
  ...trigger
}: { total: number; locked: boolean } & React.ComponentProps<'button'>) {
  return (
    <button
      type="button"
      aria-label={
        locked
          ? 'Ver detalhamento dos modificadores (requer treino)'
          : 'Ver detalhamento dos modificadores'
      }
      className={cn(
        'relative flex size-11 shrink-0 items-center justify-center rounded-lg border font-mono text-lg font-bold transition-colors hover:brightness-110',
        locked
          ? 'border-dashed border-border text-muted-foreground/50'
          : ['border-border bg-muted', accentStrong],
        className,
      )}
      {...trigger}
    >
      {signed(total)}
      {locked && (
        <Lock className="absolute -right-1 -top-1 size-3.5 rounded-full bg-background p-0.5 text-amber-500" />
      )}
    </button>
  )
}

/** Small self-labeling breakdown chip (½lvl / treino). */
function Chip({ label, value }: { label: string; value: string }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2 py-0.5 font-mono text-[11px] text-foreground   ">
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
            // Filled green vs faint outline — the old muted-vs-muted pair was
            // indistinguishable at a glance (2026-08 owner report).
            trained
              ? 'border-emerald-500/50 bg-emerald-500/15 text-emerald-600 shadow-sm dark:text-emerald-300'
              : 'border-border text-muted-foreground/50 hover:border-emerald-500/40 hover:text-muted-foreground',
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
    <ConfirmDialog
      title={`Remover ofício "${name}"?`}
      confirmLabel="Remover"
      onConfirm={onDelete}
      trigger={
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-7 text-foreground hover:bg-red-100 hover:text-red-700  dark:hover:bg-red-950/40 dark:hover:text-red-400"
          aria-label={`Remover ${name}`}
        >
          <Trash2 className="size-3.5" />
        </Button>
      }
    />
  )
}

type ItemContributions = BreakdownContribution[]

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
          'border-border bg-muted text-foreground   ',
        )}
      >
        <DialogHeader>
          <DialogTitle className={cn(accentStrong)}>
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
              note={c.note}
              indented
            />
          ))}
          <div
            className={cn(
              'mt-2 flex items-center justify-between rounded-lg border px-3 py-2',
              'border-border bg-muted  ',
            )}
          >
            <span
              className={cn('text-xs uppercase tracking-widest', subtleText)}
            >
              Total
            </span>
            <span
              className={cn(
                'flex items-center gap-1.5 font-mono text-2xl font-bold',
                locked ? 'text-muted-foreground/50' : accentStrong,
              )}
            >
              {locked && <Lock className="size-4 text-amber-500" />}
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
  note,
  indented,
}: {
  label: string
  value: number
  /** Modifier note — the WHY ("desbalanceada: -2 em ataque"), dim sub-line. */
  note?: string
  indented?: boolean
}) {
  return (
    <div
      className={cn(
        'border-b border-border py-1',
        indented && 'pl-4 text-xs opacity-80',
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="truncate">{label}</span>
        <span className="shrink-0 font-mono">{signed(value)}</span>
      </div>
      {/* wrap, never truncate: a nowrap note becomes min-content and can
          inflate the dialog past its max-width */}
      {note && (
        <p className={cn('text-[10px] leading-snug', subtleText)}>{note}</p>
      )}
    </div>
  )
}

