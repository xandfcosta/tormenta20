import { useQueryClient } from '@tanstack/solid-query'
import type { RaceAbility, RaceDefinition } from '@/shared/api/catalog-types'
import { For, Match, Show, Switch, createMemo, createSignal } from 'solid-js'
import type {
  AttributeKey,
  Character,
  RaceAttributeChoicesInput,
} from '@/shared/api/api'
import { raceAttributeMeta } from '@/shared/lib/race-attribute-meta'
import { RaceFloatingPicker, RaceSubracePicker } from '@/shared/ui/race-attribute-picker'
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

  // A escolha de atributo que a forja deixa para depois. Ela promete, no passo
  // de Resumo, "dá para criar assim e terminar na ficha" — e até a ALE-169 a
  // ficha não tinha onde terminar. A meta vem do catálogo de raças (o
  // `atributoMod`), não do `attributeBonuses` do catálogo de habilidades, que
  // só cobre as raças de modificador FIXO.
  // Acessores que estreitam, e não cast dentro do JSX: o `Match` não refina uma
  // união discriminada sozinho, e um cast aqui sobreviveria ao dia em que
  // alguém acrescentar um terceiro tipo de escolha racial. Mesmo idioma do
  // controle da forja.
  const meta = () => raceAttributeMeta(props.race.name)
  const flutuante = () => {
    const m = meta()
    return m.kind === 'floating' ? m : null
  }
  const ascendencia = () => {
    const m = meta()
    return m.kind === 'subrace' ? m : null
  }
  const escolha = () => parseRaceAttrChoice(props.character.raceAttributeChoices)

  const salvar = async (next: RaceAttributeChoicesInput) => {
    setPending(true)
    try {
      await choiceActions(queryClient, props.character.id).setRaceAttributeChoices(next)
    } catch {
      // choiceActions já reverteu e avisou o jogador.
    } finally {
      setPending(false)
    }
  }

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
      <Switch>
        <Match when={flutuante()} keyed>
          {(m) => (
            <div class="mb-2">
              <RaceFloatingPicker
                count={m.count}
                value={m.value}
                exclude={m.exclude}
                penalty={m.penalty}
                picks={(escolha().floatingPicks ?? []) as AttributeKey[]}
                onChange={(floatingPicks) =>
                  void salvar({ ...escolha(), floatingPicks })
                }
              />
            </div>
          )}
        </Match>
        <Match when={ascendencia()} keyed>
          {(m) => (
            <div class="mb-2">
              <RaceSubracePicker
                options={m.options}
                value={escolha().ascendencia}
                onChange={(ascendencia) => void salvar({ ...escolha(), ascendencia })}
              />
            </div>
          )}
        </Match>
      </Switch>

      <ul class="space-y-2">
        <For each={props.race.abilities}>
          {(ability) => (
            <li class="rounded-none border border-border p-2">
              <p class="text-xs font-semibold text-grimorio-gold">{ability.name}</p>
              <p class="mt-0.5 text-2xs leading-snug text-muted-foreground">
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
              'rounded-none border border-border px-2 py-0.5 text-2xs transition-colors',
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

/**
 * A escolha de atributo guardada no personagem. Tolerante por desenho: um blob
 * inválido vira "nada escolhido", que é o mesmo que o motor entende.
 */
function parseRaceAttrChoice(raw: string): RaceAttributeChoicesInput {
  try {
    const p = JSON.parse(raw) as { floatingPicks?: unknown; ascendencia?: unknown }
    return {
      floatingPicks: Array.isArray(p.floatingPicks)
        ? (p.floatingPicks.filter((x) => typeof x === 'string') as AttributeKey[])
        : [],
      ascendencia: typeof p.ascendencia === 'string' ? p.ascendencia : undefined,
    }
  } catch {
    return { floatingPicks: [] }
  }
}
