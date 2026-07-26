import { useMutation, useQueryClient } from '@tanstack/react-query'
import type {
  RaceAbility,
  RaceDefinition,
} from '@tormenta20/t20-data'
import { api, type Character, type AbilityChoicesResult } from '@/shared/api/api'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import type { AttributeKey } from '@/shared/api/api'
import { characterQueryOptions } from '@/entities/character/queries'
import { accentTitle, subtleText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import {
  CollapsibleAbilityCard,
  type CardFocus,
} from './collapsible-ability-card'
import { FactChips } from './fact-chips'
import { parseChoices } from './parse-choices'

const RACE_ATTR_ABBR: Record<AttributeKey, string> = {
  strength: 'For',
  dexterity: 'Des',
  constitution: 'Con',
  intelligence: 'Int',
  wisdom: 'Sab',
  charisma: 'Car',
}

function formatAttributeBonuses(
  bonuses: Partial<Record<AttributeKey, number>>,
): string {
  const parts: string[] = []
  for (const [attr, amount] of Object.entries(bonuses)) {
    if (typeof amount !== 'number' || amount === 0) continue
    const sign = amount > 0 ? '+' : ''
    parts.push(`${RACE_ATTR_ABBR[attr as AttributeKey]} ${sign}${amount}`)
  }
  return parts.join(', ')
}

/**
 * Renders the abilities granted by a single race, including variant
 * pickers for abilities that offer sub-choices (e.g. Humano's
 * `versatil` slot). Attribute bonuses derived from the race show as a
 * one-liner at the top so the sheet-header numbers stay explainable.
 */
export function RaceAbilitySection({
  race,
  character,
  focus,
  pending,
}: {
  race: RaceDefinition
  character: Character
  focus: CardFocus
  pending: number
}) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const choices = parseChoices(character.raceAbilityChoices)

  const update = useMutation<
    AbilityChoicesResult,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateAbilityChoices(character.id, {
        raceAbilityChoices: next,
      }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, raceAbilityChoices: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (delta) => {
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, ...delta } : prev,
      )
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const pickVariant = (ability: RaceAbility, variantId: string) => {
    const siblingIds = new Set(ability.variants?.map((v) => v.id) ?? [])
    const next = choices.filter((c) => !siblingIds.has(c))
    next.push(variantId)
    update.mutate(next)
  }

  const bonusLine = formatAttributeBonuses(race.attributeBonuses)
  return (
    <CollapsibleAbilityCard
      id={`raca:${race.id}`}
      title={`Raça: ${race.name}`}
      pending={pending}
      focus={focus}
    >
      {bonusLine && (
        <p className={cn('mb-2 text-xs', subtleText)}>
          <span className="font-semibold">Modificadores:</span> {bonusLine}
        </p>
      )}
      <ul className="space-y-2">
        {race.abilities.map((ability) => (
          <li key={ability.id} className="rounded border border-border p-2 ">
            <p className={cn('text-xs font-semibold', accentTitle)}>{ability.name}</p>
            <p className={cn('mt-0.5 text-[11px] leading-snug', subtleText)}>
              {ability.description}
            </p>
            <FactChips facts={ability.facts ?? []} className="mt-1" />
            {ability.variants && (
              <RaceVariantPicker
                ability={ability}
                selected={ability.variants.find((v) => choices.includes(v.id))?.id}
                onPick={(id) => pickVariant(ability, id)}
                disabled={update.isPending}
              />
            )}
          </li>
        ))}
      </ul>
    </CollapsibleAbilityCard>
  )
}

function RaceVariantPicker({
  ability,
  selected,
  onPick,
  disabled,
}: {
  ability: RaceAbility
  selected: string | undefined
  onPick: (variantId: string) => void
  disabled: boolean
}) {
  return (
    <div className="mt-2 flex flex-wrap gap-1">
      {ability.variants?.map((variant) => {
        const active = variant.id === selected
        return (
          <button
            key={variant.id}
            type="button"
            disabled={disabled}
            onClick={() => onPick(variant.id)}
            title={variant.description}
            className={cn(
              'rounded border px-2 py-0.5 text-[11px] transition-colors',
              active
                ? 'border-border bg-muted font-semibold text-foreground   '
                : 'border-border text-foreground hover:bg-muted   dark:hover:bg-muted',
              disabled && 'cursor-not-allowed opacity-60',
            )}
          >
            {variant.name}
          </button>
        )
      })}
    </div>
  )
}
