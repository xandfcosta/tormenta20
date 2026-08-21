import { SPELL_SCHOOLS } from '@/shared/api/spell-types'
import { SPELLCASTER_CLASSES } from '@/shared/rules/class-spellcasting'
import type { CatalogSpell, SpellClassName } from '@/shared/api/catalog-types'
import type { SpellCircle, SpellSchool } from '@/shared/api/spell-types'
import { type AttributeKey } from '@/shared/api/attribute-keys'
import { BookOpen, Plus, Search } from 'lucide-solid'
import { For, Show, createMemo, createSignal } from 'solid-js'
import { computedSheetFor } from '@/entities/character/computed-sheet'
import { grantedSpells } from '@/entities/character/granted-spells'
import { castableClassesFor } from '@/entities/character/spell-rules'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import { normalize } from './normalize'
import { CIRCLE_LABEL, SCHOOL_LABEL } from './spell-labels'
import { SpellRow } from './spell-row'

const CIRCLES: readonly SpellCircle[] = [0, 1, 2, 3, 4, 5]

const SELECT_CLASS =
  'h-8 cursor-pointer rounded-sm border border-input bg-transparent px-2 text-sm outline-none focus:ring-2 focus:ring-ring'

/**
 * The character's grimoire: only the spells LEARNED, each with
 * prepare/unlearn/cast. "Aprender" opens a dialog over the whole Cap 4 catalog
 * with círculo/escola/classe filters. Persistence lives in the `CharacterSpell`
 * join rows on `character.spells`.
 */
export function SpellbookPanel(props: { character: Character }) {
  const conditionals = useConditionals()

  // Computed once for the whole grimoire — every row reads the same
  // per-attribute CD map the sheet's "CD Magia" box uses.
  const spellCdByAttribute = createMemo(
    () =>
      computedSheetFor(props.character, conditionals.active(props.character.id))
        .spellCdByAttribute,
  )
  const casterClasses = createMemo(() =>
    castableClassesFor(props.character, SPELLCASTER_CLASSES),
  )
  const learnedById = createMemo(() => {
    const byId = new Map<string, CharacterSpell>()
    for (const spell of props.character.spells) byId.set(spell.catalogSpellId, spell)
    return byId
  })
  const learned = createMemo(() => {
    const catalog = spellCatalog()
    return props.character.spells
      .map((spell) => catalog[spell.catalogSpellId])
      .filter((spell): spell is CatalogSpell => Boolean(spell))
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  })
  // Powers that teach a spell (Totem Espiritual, p42) — visible even to a
  // non-caster: a Bárbaro with Totem has to see his spell.
  const granted = createMemo(() => grantedSpells(props.character))

  return (
    <section class="flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-none border border-grimorio-iron bg-grimorio-panel">
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <div class="flex items-baseline gap-3">
          <h2 class="flex items-center gap-2 font-heading text-lg uppercase tracking-wide text-grimorio-gold">
            <BookOpen aria-hidden="true" class="size-4" />
            Grimório
          </h2>
          <p class="text-[10px] text-muted-foreground sm:text-xs">
            {learned().length} aprendida{learned().length === 1 ? '' : 's'}
          </p>
        </div>
        <Show when={casterClasses().length > 0}>
          <LearnSpellDialog
            character={props.character}
            learnedById={learnedById()}
            spellCdByAttribute={spellCdByAttribute()}
          />
        </Show>
      </div>

      <Show when={granted().length > 0}>
        <div class="shrink-0 space-y-1 border-b border-grimorio-iron px-2 py-1">
          <p class="px-2 pt-1 text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
            Concedidas por poderes
          </p>
          <For each={granted()}>
            {(entry) => (
              <SpellRow
                spell={entry.spell}
                character={props.character}
                learned={learnedById().get(entry.spell.id) ?? null}
                spellCdByAttribute={spellCdByAttribute()}
                granted={{ sourcePower: entry.sourcePower, keyAttribute: entry.keyAttribute }}
              />
            )}
          </For>
        </div>
      </Show>

      <Show
        when={casterClasses().length > 0}
        fallback={
          <Show when={granted().length === 0}>
            <p class="px-4 py-3 text-[11px] text-muted-foreground">
              Este personagem não tem classe conjuradora.
            </p>
          </Show>
        }
      >
        <Show
          when={learned().length > 0}
          fallback={
            <p class="min-h-0 flex-1 px-4 py-6 text-center text-xs text-muted-foreground">
              Nenhuma magia aprendida. Use "Aprender" para adicionar magias.
            </p>
          }
        >
          <VirtualList
            class="min-h-0 flex-1 px-2 py-1"
            items={learned()}
            estimateSize={44}
            getKey={(spell) => spell.id}
            renderItem={(spell) => (
              <SpellRow
                spell={spell}
                character={props.character}
                learned={learnedById().get(spell.id) ?? null}
                spellCdByAttribute={spellCdByAttribute()}
              />
            )}
          />
        </Show>
      </Show>
    </section>
  )
}

/**
 * Full-catalog learn dialog: search plus círculo/escola/classe filters over the
 * whole spell catalog. Each row exposes the Learn action, and spells already
 * known show as such, so the list doubles as a browse view.
 */
function LearnSpellDialog(props: {
  character: Character
  learnedById: Map<string, CharacterSpell>
  spellCdByAttribute: Record<AttributeKey, number>
}) {
  const [open, setOpen] = createSignal(false)
  const [query, setQuery] = createSignal('')
  const [circle, setCircle] = createSignal<SpellCircle | 'all'>('all')
  const [school, setSchool] = createSignal<SpellSchool | 'all'>('all')
  const [classFilter, setClassFilter] = createSignal<SpellClassName | 'all'>('all')

  const filtered = createMemo(() => {
    const search = query().trim() ? normalize(query()) : ''
    return Object.values(spellCatalog())
      .filter((spell) => {
        if (circle() !== 'all' && spell.circle !== circle()) return false
        if (school() !== 'all' && spell.school !== school()) return false
        if (classFilter() !== 'all' && !spell.classes.includes(classFilter() as SpellClassName)) {
          return false
        }
        return !search || normalize(spell.name).includes(search)
      })
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  })

  return (
    <>
      <Button type="button" size="sm" class="h-7 gap-1 text-xs" onClick={() => setOpen(true)}>
        <Plus aria-hidden="true" class="size-3.5" />
        Aprender
      </Button>

      <Dialog open={open()} onOpenChange={setOpen}>
        <DialogContent class="flex max-h-[85vh] w-[calc(100vw-1.5rem)] max-w-2xl flex-col gap-3">
          <DialogHeader>
            <DialogTitle>Aprender magia</DialogTitle>
          </DialogHeader>

          <div class="grid shrink-0 gap-2 sm:grid-cols-4">
            <div class="relative sm:col-span-2">
              <Search
                aria-hidden="true"
                class="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
              />
              <Input
                type="search"
                value={query()}
                onInput={(event) => setQuery(event.currentTarget.value)}
                placeholder="Buscar magia"
                aria-label="Buscar magia"
                class="h-8 pl-7 text-sm"
              />
            </div>
            <select
              value={circle() === 'all' ? '' : String(circle())}
              onChange={(event) =>
                setCircle(
                  event.currentTarget.value === ''
                    ? 'all'
                    : (Number(event.currentTarget.value) as SpellCircle),
                )
              }
              aria-label="Círculo"
              class={SELECT_CLASS}
            >
              <option value="">Todos os círculos</option>
              <For each={CIRCLES}>
                {(value) => <option value={value}>{CIRCLE_LABEL[value]}</option>}
              </For>
            </select>
            <select
              value={school() === 'all' ? '' : school()}
              onChange={(event) =>
                setSchool(
                  event.currentTarget.value === ''
                    ? 'all'
                    : (event.currentTarget.value as SpellSchool),
                )
              }
              aria-label="Escola"
              class={SELECT_CLASS}
            >
              <option value="">Todas as escolas</option>
              <For each={SPELL_SCHOOLS}>
                {(value) => <option value={value}>{SCHOOL_LABEL[value]}</option>}
              </For>
            </select>
            <select
              value={classFilter() === 'all' ? '' : classFilter()}
              onChange={(event) =>
                setClassFilter(
                  event.currentTarget.value === ''
                    ? 'all'
                    : (event.currentTarget.value as SpellClassName),
                )
              }
              aria-label="Lista de classe"
              class={`${SELECT_CLASS} sm:col-span-4`}
            >
              <option value="">Todas as classes</option>
              <For each={SPELLCASTER_CLASSES}>{(value) => <option value={value}>{value}</option>}</For>
            </select>
          </div>

          <Show
            when={filtered().length > 0}
            fallback={
              <p class="min-h-0 flex-1 px-2 py-6 text-center text-sm text-muted-foreground">
                Nenhuma magia para "{query()}"
              </p>
            }
          >
            <VirtualList
              class="min-h-0 flex-1 px-1 py-1"
              items={filtered()}
              estimateSize={44}
              getKey={(spell) => spell.id}
              renderItem={(spell) => (
                <SpellRow
                  spell={spell}
                  character={props.character}
                  learned={props.learnedById.get(spell.id) ?? null}
                  spellCdByAttribute={props.spellCdByAttribute}
                />
              )}
            />
          </Show>
        </DialogContent>
      </Dialog>
    </>
  )
}
