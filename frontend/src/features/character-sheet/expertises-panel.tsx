import { useQueryClient } from '@tanstack/solid-query'
import { Search } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import {
  ATTRIBUTE_ABBR,
  EXPERTISES,
  type ExpertiseDef,
  trainingBonusForLevel,
} from '@/entities/character/expertise'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import type { Character } from '@/shared/api/api'
import { Input } from '@/shared/ui/input'
import { AddCustomExpertiseDialog } from './add-custom-expertise-dialog'
import { expertiseActions } from './expertise-mutations'
import { ExpertiseRow } from './expertise-row'
import { normalize } from './normalize'
import type { SheetPanelProps } from './sheet-sections'

// Resistências first — "teste de Reflexos!" is the hottest lookup at the table,
// so they are pinned above the alphabetical rest.
const RESISTENCIAS = ['Fortitude', 'Reflexos', 'Vontade']

/** Standard perícias (resistências pinned) plus the character's own ofícios. */
export function expertiseDefsFor(character: Character): ExpertiseDef[] {
  const custom: ExpertiseDef[] = character.expertises
    .filter((e) => e.custom)
    .map((e) => ({
      name: e.name,
      attribute: e.attribute,
      abbr: ATTRIBUTE_ABBR[e.attribute],
      trainedOnly: true,
    }))
  const pinned = EXPERTISES.filter((e) => RESISTENCIAS.includes(e.name))
  const rest = EXPERTISES.filter((e) => !RESISTENCIAS.includes(e.name))
  return [...pinned, ...rest, ...custom]
}

/** Accent-insensitive name filter — "pericia" must find "Perícia". */
export function filterExpertises(defs: ExpertiseDef[], query: string): ExpertiseDef[] {
  if (query.trim() === '') return defs
  return defs.filter((d) => normalize(d.name).includes(normalize(query)))
}

/**
 * The Perícias block: every skill with its total, training toggle and the
 * breakdown behind it.
 *
 * This is the heaviest block in the sheet (~850 DOM nodes) — the one the React
 * app wrapped in `memo()` so a tab switch would not re-run it. There is no
 * wrapper here; a switch touches only what reads the signal that changed.
 */
export function ExpertisesPanel(props: SheetPanelProps) {
  const [query, setQuery] = createSignal('')
  const queryClient = useQueryClient()
  const actions = () => expertiseActions(queryClient, props.character.id)

  // The conditionals store lands with the Efeitos block (ALE-86); until then no
  // opt-in effect is toggled, so the sheet computes against an empty set.
  const sheet = createMemo(() => computedSheetFor(props.character, new Set<string>()))
  const defs = createMemo(() => expertiseDefsFor(props.character))
  const visible = createMemo(() => filterExpertises(defs(), query()))

  const isCustom = (def: ExpertiseDef) => !EXPERTISES.some((b) => b.name === def.name)

  return (
    <section class="flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-none border border-grimorio-iron bg-grimorio-panel">
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <div class="flex items-baseline gap-3">
          <h2 class="font-heading text-lg uppercase tracking-wide text-grimorio-gold">Perícias</h2>
          <p class="text-3xs text-muted-foreground sm:text-xs">
            treino +{trainingBonusForLevel(props.character.level)} • ½ nível{' '}
            {Math.floor(props.character.level / 2)}
          </p>
        </div>
        <div class="flex w-full items-center gap-2 sm:w-auto">
          <div class="relative flex-1 sm:w-56 sm:flex-none">
            <Search
              aria-hidden="true"
              class="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
            />
            <Input
              type="search"
              value={query()}
              onInput={(event) => setQuery(event.currentTarget.value)}
              placeholder="Buscar perícia"
              aria-label="Buscar perícia"
              class="h-7 pl-7 text-xs"
            />
          </div>
          <AddCustomExpertiseDialog onAdd={(input) => actions().addCustom(input)} />
        </div>
      </div>

      {/* @container e não `xl:`: a largura que decide o número de colunas é a
          desta lista, não a da janela. Com `xl:grid-cols-2`, uma janela de 1920
          quebrava em duas colunas DENTRO do painel do combatente (518px), dando
          240px por cartão — e o nome da perícia, que é `flex-1 truncate`, era
          espremido até desaparecer: sobravam o número e os controles, sem dizer
          de QUE perícia. Mesma família da ALE-122. */}
      <div class="@container min-h-0 flex-1 overflow-auto px-2 py-1">
        <Show
          when={visible().length > 0}
          fallback={
            <p class="px-2 py-3 text-center text-xs text-muted-foreground">
              Nenhuma perícia para "{query()}"
            </p>
          }
        >
          <div class="grid gap-2 @[44rem]:grid-cols-2">
            <For each={visible()}>
              {(def) => (
                <ExpertiseRow
                  character={props.character}
                  def={def}
                  sheet={sheet()}
                  glance={props.glance}
                  onPatch={(patch) => actions().update(def.name, patch)}
                  onDelete={isCustom(def) ? () => actions().remove(def.name) : undefined}
                />
              )}
            </For>
          </div>
        </Show>
      </div>
    </section>
  )
}
