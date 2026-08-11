import { useQueryClient } from '@tanstack/solid-query'
import type { RaceAbility, RaceDefinition } from '@tormenta20/t20-data'
import { For, Show, createMemo, createSignal } from 'solid-js'
import type { AttributeKey, Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { choiceActions } from './choice-mutations'
import { pickExclusive } from './choice-lists'
import { type CardFocus, CollapsibleAbilityCard } from './collapsible-ability-card'
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

/** "For +2, Des +1" — keeps the header's attribute numbers explainable. */
function formatAttributeBonuses(bonuses: Partial<Record<AttributeKey, number>>): string {
  return Object.entries(bonuses)
    .filter(([, amount]) => typeof amount === 'number' && amount !== 0)
    .map(([attr, amount]) => {
      const sign = (amount as number) > 0 ? '+' : ''
      return `${RACE_ATTR_ABBR[attr as AttributeKey]} ${sign}${amount}`
    })
    .join(', ')
}

/**
 * The abilities one race grants, including variant pickers for the abilities
 * that offer sub-choices (Humano's `versatil` slot). Attribute bonuses show as
 * a one-liner at the top.
 */
export function RaceAbilitySection(props: {
  race: RaceDefinition
  character: Character
  focus: CardFocus
  pending: number
}) {
  const queryClient = useQueryClient()
  const [pending, setPending] = createSignal(false)
  const choices = createMemo(() => parseChoices(props.character.raceAbilityChoices))

  const pickVariant = async (ability: RaceAbility, variantId: string) => {
    const siblings = new Set(ability.variants?.map((v) => v.id) ?? [])
    setPending(true)
    try {
      await choiceActions(queryClient, props.character.id).setRaceAbilityChoices(
        pickExclusive(choices(), siblings, variantId),
      )
    } catch {
      // choiceActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  const bonusLine = () => formatAttributeBonuses(props.race.attributeBonuses)

  return (
    <CollapsibleAbilityCard
      id={`raca:${props.race.id}`}
      title={`Raça: ${props.race.name}`}
      pending={props.pending}
      focus={props.focus}
    >
      <Show when={bonusLine()}>
        {(line) => (
          <p class="mb-2 text-xs text-muted-foreground">
            <span class="font-semibold">Modificadores:</span> {line()}
          </p>
        )}
      </Show>
      <ul class="space-y-2">
        <For each={props.race.abilities}>
          {(ability) => (
            <li class="rounded-sm border border-border p-2">
              <p class="text-xs font-semibold text-grimorio-gold">{ability.name}</p>
              <p class="mt-0.5 text-[11px] leading-snug text-muted-foreground">
                {ability.description}
              </p>
              <FactChips facts={ability.facts ?? []} class="mt-1" />
              <Show when={ability.variants}>
                {(variants) => (
                  <RaceVariantPicker
                    variants={variants()}
                    selected={variants().find((v) => choices().includes(v.id))?.id}
                    disabled={pending()}
                    onPick={(id) => void pickVariant(ability, id)}
                  />
                )}
              </Show>
            </li>
          )}
        </For>
      </ul>
    </CollapsibleAbilityCard>
  )
}

function RaceVariantPicker(props: {
  variants: NonNullable<RaceAbility['variants']>
  selected: string | undefined
  disabled: boolean
  onPick: (variantId: string) => void
}) {
  return (
    <div class="mt-2 flex flex-wrap gap-1">
      <For each={props.variants}>
        {(variant) => (
          <button
            type="button"
            disabled={props.disabled}
            onClick={() => props.onPick(variant.id)}
            title={variant.description}
            aria-pressed={variant.id === props.selected}
            class={cn(
              'rounded-sm border border-border px-2 py-0.5 text-[11px] transition-colors',
              variant.id === props.selected
                ? 'bg-muted font-semibold text-foreground'
                : 'text-foreground hover:bg-muted',
              props.disabled && 'cursor-not-allowed opacity-60',
            )}
          >
            {variant.name}
          </button>
        )}
      </For>
    </div>
  )
}
