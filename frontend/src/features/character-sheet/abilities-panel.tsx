import type { RaceDefinition } from '@/shared/api/catalog-types'
import { Settings2 } from 'lucide-solid'
import { For, type JSX, Show, createMemo, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { getRace } from '@/shared/lib/abilities-cache'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { cn } from '@/shared/lib/utils'
import { ClassesSection } from './class-abilities'
import type { CardFocus } from './collapsible-ability-card'
import { OriginAbilitySection } from './origin-abilities'
import { type Pendencia, type PendenciaSource, computePendencias } from './pendencias'
import { PendenciasCallout } from './pendencias-callout'
import { PowerActionSlot } from './power-action-slot'
import { PowerPlayList } from './power-play-list'
import { ownedPowerSpec } from './power-spec-resolver'
import { RaceAbilitySection } from './race-abilities'
import { normalize } from './normalize'
import { ownedAbilities } from './sheet-search-index'

const SOURCE_TABS: { value: PendenciaSource; label: string }[] = [
  { value: 'raca', label: 'Raça' },
  { value: 'origem', label: 'Origem' },
  { value: 'classe', label: 'Classe' },
]

/**
 * The Poderes block — a Pendências callout over three source sub-tabs (Raça /
 * Origem / Classe), each rendering its abilities as collapsible cards. Clicking
 * a pendência jumps to its tab and opens the owning card.
 *
 * Two modes: `play` is the table-side list of what you can DO, `edit` is the
 * acquisition UI (checkboxes, pickers, pendências). It opens in `edit` while
 * choices are still owed, so onboarding is never hidden behind a toggle.
 */
export function AbilitiesPanel(props: { character: Character }) {
  const pendencias = createMemo(() => computePendencias(props.character))
  const pendingByCard = createMemo(() => {
    const byCard = new Map<string, number>()
    for (const pendencia of pendencias()) {
      byCard.set(pendencia.cardId, (byCard.get(pendencia.cardId) ?? 0) + 1)
    }
    return byCard
  })

  // Opens on the first source that still owes a choice; falls back to Raça.
  const [tab, setTab] = createSignal<PendenciaSource>(pendencias()[0]?.source ?? 'raca')
  const [focus, setFocus] = createSignal<CardFocus>(null)
  const [query, setQuery] = createSignal('')
  const [mode, setMode] = createSignal<'play' | 'edit'>(
    pendencias().length > 0 ? 'edit' : 'play',
  )

  const races = createMemo(() =>
    props.character.races
      .map((entry) => getRace(entry.race))
      .filter((race): race is RaceDefinition => Boolean(race)),
  )

  const jump = (pendencia: Pendencia) => {
    setMode('edit')
    setTab(pendencia.source)
    // A fresh nonce so re-clicking the same pendência re-opens the card.
    setFocus({ id: pendencia.cardId, nonce: focus() ? (focus()?.nonce ?? 0) + 1 : 1 })
  }

  const countFor = (source: PendenciaSource) =>
    pendencias().filter((pendencia) => pendencia.source === source).length

  return (
    <section class="flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-none border border-grimorio-iron bg-grimorio-panel">
      <div class="shrink-0 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <h2 class="font-heading text-lg uppercase tracking-wide text-grimorio-gold">Poderes</h2>
      </div>

      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-3 sm:p-4">
        <Show when={mode() === 'edit'}>
          <PendenciasCallout pendencias={pendencias()} onJump={jump} />
        </Show>

        {/* Flat lookup by NAME: at the table the player knows the power's name,
            not which source granted it. */}
        <div class="flex shrink-0 items-center gap-2">
          <Input
            value={query()}
            onInput={(event) => setQuery(event.currentTarget.value)}
            placeholder="Buscar poder ou habilidade…"
            aria-label="Buscar poder ou habilidade"
            class="h-8 min-w-0 flex-1 text-xs"
          />
          <Button
            type="button"
            variant={mode() === 'edit' ? 'default' : 'outline'}
            size="sm"
            class="h-8 shrink-0 gap-1 text-xs"
            onClick={() => setMode(mode() === 'edit' ? 'play' : 'edit')}
          >
            <Settings2 aria-hidden="true" class="size-3.5" />
            {mode() === 'edit' ? 'Voltar ao jogo' : 'Editar poderes'}
            <Show when={mode() === 'play' && pendencias().length > 0}>
              <span class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold leading-none text-white">
                {pendencias().length}
              </span>
            </Show>
          </Button>
        </div>

        <Show
          when={query().trim() === ''}
          fallback={<FlatAbilityResults character={props.character} query={query()} />}
        >
          <Show when={mode() === 'edit'} fallback={<PowerPlayList character={props.character} />}>
            <SourceTabs
              active={tab()}
              countFor={countFor}
              onPick={setTab}
            />
            <div class="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
              <Show when={tab() === 'raca'}>
                <Show
                  when={races().length > 0}
                  fallback={<EmptyHint>Raça do personagem não está no catálogo.</EmptyHint>}
                >
                  <For each={races()}>
                    {(race) => (
                      <RaceAbilitySection
                        race={race}
                        character={props.character}
                        focus={focus()}
                        pending={pendingByCard().get(`raca:${race.id}`) ?? 0}
                      />
                    )}
                  </For>
                </Show>
              </Show>

              <Show when={tab() === 'origem'}>
                <OriginAbilitySection
                  character={props.character}
                  focus={focus()}
                  pending={pendingByCard().get('origem') ?? 0}
                />
              </Show>

              <Show when={tab() === 'classe'}>
                <Show
                  when={props.character.classes.length > 0}
                  fallback={<EmptyHint>Nenhuma classe atribuída.</EmptyHint>}
                >
                  <For each={props.character.classes}>
                    {(entry) => (
                      <ClassesSection
                        entry={entry}
                        character={props.character}
                        focus={focus()}
                        pending={pendingByCard().get(`classe:${entry.className}`) ?? 0}
                      />
                    )}
                  </For>
                </Show>
              </Show>
            </div>
          </Show>
        </Show>
      </div>
    </section>
  )
}

/**
 * Custom pill row instead of the shared Tabs primitive: this panel is nested
 * inside the sheet's VERTICAL tabs, and a nested TabsList inherits that
 * orientation and stacks.
 */
function SourceTabs(props: {
  active: PendenciaSource
  countFor: (source: PendenciaSource) => number
  onPick: (source: PendenciaSource) => void
}) {
  return (
    // overflow-y-hidden: a lone `overflow-x-auto` lets the browser promote
    // overflow-y to `auto`, and the 1px of the active tab's border spawned a
    // spurious vertical scrollbar on this single-row strip.
    <div class="flex shrink-0 gap-1 overflow-x-auto overflow-y-hidden border-b border-grimorio-iron">
      <For each={SOURCE_TABS}>
        {(source) => (
          <button
            type="button"
            onClick={() => props.onPick(source.value)}
            aria-pressed={props.active === source.value}
            class={cn(
              '-mb-px flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-1.5 text-sm font-medium transition-colors',
              props.active === source.value
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            {source.label}
            <Show when={props.countFor(source.value) > 0}>
              <span class="inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-destructive px-1 text-[10px] font-bold leading-none text-white">
                {props.countFor(source.value)}
              </span>
            </Show>
          </button>
        )}
      </For>
    </div>
  )
}

/** Flat, source-badged results for the abilities search. */
function FlatAbilityResults(props: { character: Character; query: string }) {
  // normalize: accent-insensitive ("furia" finds "Fúria"), same helper as the
  // sheet's other searches.
  const results = createMemo(() => {
    const search = normalize(props.query.trim())
    return ownedAbilities(props.character).filter((entry) =>
      normalize(entry.name).includes(search),
    )
  })

  return (
    <Show
      when={results().length > 0}
      fallback={<EmptyHint>Nenhum poder para "{props.query}".</EmptyHint>}
    >
      <ul class="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
        <For each={results()}>
          {(entry) => (
            <li class="rounded-none border border-border p-2">
              <div class="flex flex-wrap items-center gap-1.5">
                <p class="text-xs font-semibold">{entry.name}</p>
                <span class="rounded-none bg-muted px-1 py-0 text-[9px] text-muted-foreground">
                  {entry.source}
                </span>
                <PowerActionSlot
                  spec={ownedPowerSpec(entry)}
                  character={props.character}
                  class="ml-auto"
                />
              </div>
              <p class="mt-0.5 text-[11px] leading-snug text-muted-foreground">{entry.detail}</p>
            </li>
          )}
        </For>
      </ul>
    </Show>
  )
}

function EmptyHint(props: { children: JSX.Element }) {
  return <p class="text-xs italic text-muted-foreground">{props.children}</p>
}
