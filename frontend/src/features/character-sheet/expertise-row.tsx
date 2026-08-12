import { Dumbbell, Star, Trash2 } from 'lucide-solid'
import { For, Show } from 'solid-js'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
  type ExpertiseDef,
  expertiseStateFor,
  trainingBonusForLevel,
} from '@/entities/character/expertise'
import { expertiseFromSheet } from '@/entities/character/computed-sheet'
import type { AttributeKey, Character } from '@/shared/api/api'
import type { ComputedSheetV2 } from '@/shared/lib/computed-sheet-v2'
import { cn } from '@/shared/lib/utils'
import { DialogTrigger } from '@/shared/ui/dialog'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/shared/ui/tooltip'
import { ExpertiseBreakdown } from './expertise-breakdown'
import type { ExpertisePatch } from './expertise-mutations'
import { signed } from './signed'

export type ExpertiseRowProps = {
  character: Character
  def: ExpertiseDef
  sheet: ComputedSheetV2
  onPatch: (patch: ExpertisePatch) => void
  /** Only custom "ofícios" can be deleted. */
  onDelete?: () => void
}

/**
 * One perícia: its total (which opens the breakdown), the name, the trained
 * toggle, the attribute it keys off, and the chips that preview the math.
 */
export function ExpertiseRow(props: ExpertiseRowProps) {
  const state = () => expertiseStateFor(props.character, props.def)
  // Every standard + custom perícia is on the sheet; `?? 0` is only a type guard.
  const entry = () => expertiseFromSheet(props.sheet, props.def.name)
  const total = () => entry()?.total ?? 0
  const halfLevel = () => Math.floor(props.character.level / 2)
  const trainBonus = () => (state().trained ? trainingBonusForLevel(props.character.level) : 0)
  const itemBonus = () => entry()?.itemBonus ?? 0
  const locked = () => !!props.def.trainedOnly && !state().trained

  return (
    <ExpertiseBreakdown
      name={props.def.name}
      total={total()}
      locked={locked()}
      halfLevel={halfLevel()}
      attrAbbr={ATTRIBUTE_ABBR[state().attribute]}
      attrMod={entry()?.attrValue ?? 0}
      trainBonus={trainBonus()}
      itemBonus={itemBonus()}
      contributions={entry()?.itemContributions ?? []}
    >
      <div
        class={cn(
          'flex items-start gap-2.5 rounded-sm border border-grimorio-iron p-2.5 transition-colors hover:border-grimorio-gold/50',
          state().trained && 'bg-[var(--grimorio-panel)]',
        )}
      >
        {/* Both the badge and the name open the breakdown; the toggle, the
            attribute select and delete stay interactive — they are not
            triggers. Kobalte composes via `as=`, where Radix used `asChild`. */}
        <DialogTrigger as={TotalBadge} total={total()} locked={locked()} />
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-1.5">
            <DialogTrigger
              as="button"
              type="button"
              class="min-w-0 flex-1 truncate text-left text-sm text-foreground hover:underline"
            >
              {props.def.name}
            </DialogTrigger>
            <Show when={props.def.trainedOnly}>
              <TrainedOnlyStar locked={locked()} />
            </Show>
            <TrainedToggle
              trained={state().trained}
              name={props.def.name}
              onToggle={(next) => props.onPatch({ trained: next })}
            />
            <Show when={props.onDelete}>
              {(onDelete) => <DeleteExpertiseButton name={props.def.name} onDelete={onDelete()} />}
            </Show>
          </div>
          <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
            <AttributeSelect
              name={props.def.name}
              value={state().attribute}
              sheet={props.sheet}
              onChange={(attribute) => props.onPatch({ attribute })}
            />
            <Chip label="½lvl" value={String(halfLevel())} />
            <Chip label="treino" value={signed(trainBonus())} />
            <DialogTrigger as="button" type="button" class="inline-flex hover:brightness-110">
              <Chip label="outros" value={signed(itemBonus())} />
            </DialogTrigger>
          </div>
        </div>
      </div>
    </ExpertiseBreakdown>
  )
}

/**
 * The perícia's number, doubling as the breakdown trigger. Locked (trained-only
 * and still untrained) reads as a dashed, dimmed box — the old line-through was
 * illegible on small mono digits.
 */
function TotalBadge(props: { total: number; locked: boolean }) {
  return (
    <button
      {...props}
      type="button"
      aria-label={`Detalhar ${signed(props.total)}`}
      class={cn(
        'flex size-11 shrink-0 items-center justify-center rounded-sm border font-mono text-lg font-bold',
        props.locked
          ? 'border-dashed border-grimorio-iron text-muted-foreground/50'
          : 'border-grimorio-iron bg-[var(--grimorio-panel-raised)] text-grimorio-gold',
      )}
    >
      {signed(props.total)}
    </button>
  )
}

/** Star marking a trained-only perícia; amber while it is locked out. */
function TrainedOnlyStar(props: { locked: boolean }) {
  return (
    <Tooltip>
      <TooltipTrigger
        as="button"
        type="button"
        aria-label="Apenas treinada"
        class="inline-flex shrink-0 cursor-help"
      >
        <Star
          aria-hidden="true"
          class={cn(
            'size-3',
            props.locked
              ? 'fill-amber-500 text-amber-500'
              : 'fill-none text-muted-foreground/60',
          )}
        />
      </TooltipTrigger>
      <TooltipContent>Pode ser usada apenas quando treinada</TooltipContent>
    </Tooltip>
  )
}

function TrainedToggle(props: { trained: boolean; name: string; onToggle: (next: boolean) => void }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={props.trained}
      aria-label={`${props.name} treinada`}
      onClick={() => props.onToggle(!props.trained)}
      class={cn(
        'inline-flex shrink-0 items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wider transition-colors',
        props.trained
          ? 'border-grimorio-gold/60 text-grimorio-gold'
          : 'border-grimorio-iron text-muted-foreground hover:border-grimorio-gold/40',
      )}
    >
      <Dumbbell aria-hidden="true" class="size-3" />
      Treino
    </button>
  )
}

function DeleteExpertiseButton(props: { name: string; onDelete: () => void }) {
  return (
    <button
      type="button"
      aria-label={`Remover ${props.name}`}
      onClick={() => props.onDelete()}
      class="inline-flex shrink-0 rounded-sm p-1 text-muted-foreground transition-colors hover:text-destructive"
    >
      <Trash2 aria-hidden="true" class="size-3.5" />
    </button>
  )
}

/**
 * Which attribute the perícia keys off. Shows the FINAL modifier (race and item
 * bonuses folded in), not the raw sheet value — otherwise the row disagrees
 * with its own breakdown and total.
 */
function AttributeSelect(props: {
  name: string
  value: AttributeKey
  sheet: ComputedSheetV2
  onChange: (attribute: AttributeKey) => void
}) {
  return (
    <select
      value={props.value}
      onChange={(event) => props.onChange(event.currentTarget.value as AttributeKey)}
      aria-label={`${props.name} atributo`}
      class="h-6 cursor-pointer rounded-full border border-grimorio-iron bg-transparent px-2 font-mono text-[11px] outline-none focus:ring-2 focus:ring-ring"
    >
      <For each={ATTRIBUTE_KEYS}>
        {(key) => (
          <option value={key}>
            {ATTRIBUTE_ABBR[key]} {signed(props.sheet.attributes[key].total)}
          </option>
        )}
      </For>
    </select>
  )
}

function Chip(props: { label: string; value: string }) {
  return (
    <span class="inline-flex items-center gap-1 rounded-full border border-grimorio-iron px-2 py-0.5 text-[10px] text-muted-foreground">
      <span class="uppercase tracking-wider">{props.label}</span>
      <span class="font-mono text-foreground">{props.value}</span>
    </span>
  )
}
