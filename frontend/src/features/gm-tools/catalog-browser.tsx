import type { CatalogSpell, Condition } from '@/shared/api/catalog-types'
import type { CatalogItem } from '@/shared/api/item-types'
import { Search } from 'lucide-solid'
import { type JSX, Match, Show, Switch, createMemo, createSignal } from 'solid-js'
import { allCatalogItems } from '@/shared/lib/catalog-cache'
import { conditionsList } from '@/shared/lib/rules-catalog-cache'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { Input } from '@/shared/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { VirtualList } from '@/shared/ui/virtual-list'
import {
  type CatalogPower,
  type CatalogResultRow,
  catalogPowers,
  catalogSearchRows,
} from './catalog-model'
import {
  ConditionRow,
  ItemCatalogRow,
  PowerCatalogRow,
  SpellCatalogRow,
} from './catalog-rows'
import { SectionLabel } from '@/shared/ui/section-label'

export type CatalogBrowserProps = {
  /** Bounds the list's scroll area — the drawer gives it less room than the
   *  Mesa's tool does. */
  listClass?: string
}

/**
 * Tabbed catalog browser (condições / magias / poderes / itens) over ONE shared
 * search box. Owns its query state so it drops unchanged into both the Mesa's
 * Catálogos tool and the in-session panel.
 *
 * A live query searches EVERY catalog at once; an empty box falls back to
 * tab-by-tab browsing. That is what makes the "todos os catálogos" placeholder
 * honest — the React version filtered only the active tab, so "bola de fogo"
 * typed on the Condições tab read "nada encontrado" while the spell existed
 * (ALE-22).
 */
export function CatalogBrowser(props: CatalogBrowserProps) {
  const [query, setQuery] = createSignal('')

  // Read INSIDE the component: the catalogs are fetched and primed by the
  // loader gate, so a module-level const would freeze an empty list (#13).
  const items = createMemo(() =>
    [...allCatalogItems()].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR')),
  )
  const powers = createMemo(() => catalogPowers())
  const spells = createMemo(() =>
    Object.values(spellCatalog()).sort(
      (a, b) => a.circle - b.circle || a.name.localeCompare(b.name, 'pt-BR'),
    ),
  )
  const conditions = createMemo(() =>
    [...conditionsList()].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR')),
  )

  const searching = () => query().trim().length > 0

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-3">
      <div class="relative shrink-0">
        <Search
          class="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          type="search"
          value={query()}
          onInput={(event) => setQuery(event.currentTarget.value)}
          placeholder="Buscar em todos os catálogos"
          aria-label="Buscar nos catálogos"
          class="pl-8"
        />
      </div>

      <Show
        when={searching()}
        fallback={
          // `flex flex-col` em cada TabsContent, e não só `min-h-0 flex-1`: o
          // painel de aba é um BLOCO, então o `flex-1` do filho não valia nada
          // e nada limitava a altura dele — a lista crescia até a altura do
          // conteúdo e vazava 1854-2566px para fora do cartão, nos seis
          // formatos, sem a página rolar (ALE-149).
          <Tabs defaultValue="conditions" class="flex min-h-0 flex-1 flex-col gap-2">
            <TabsList class="max-w-full self-start overflow-x-auto">
              <TabsTrigger value="conditions">Condições</TabsTrigger>
              <TabsTrigger value="spells">Magias</TabsTrigger>
              <TabsTrigger value="powers">Poderes</TabsTrigger>
              <TabsTrigger value="items">Itens</TabsTrigger>
            </TabsList>

            {/* `flex flex-col` em cada painel, e não só `min-h-0 flex-1`: o
                painel de aba é um BLOCO, então o `flex-1` do filho não valia
                nada e nada limitava a altura dele — a lista crescia até a
                altura do conteúdo e vazava para FORA do cartão, medido em
                1854-2566px nos seis formatos, e sem a página rolar, que é
                por que a asserção "a cena não rola" ficava verde por cima
                (ALE-149). */}
            <TabsContent value="conditions" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={conditions()}
                estimateSize={88}
                listClass={props.listClass}
                getKey={(condition) => condition.id}
                renderRow={(condition) => <ConditionRow condition={condition} />}
              />
            </TabsContent>
            <TabsContent value="spells" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={spells()}
                estimateSize={140}
                listClass={props.listClass}
                getKey={(spell) => spell.id}
                renderRow={(spell) => <SpellCatalogRow spell={spell} />}
              />
            </TabsContent>
            <TabsContent value="powers" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={powers()}
                estimateSize={100}
                listClass={props.listClass}
                getKey={(power) => power.id}
                renderRow={(power) => <PowerCatalogRow power={power} />}
              />
            </TabsContent>
            <TabsContent value="items" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={items()}
                estimateSize={84}
                listClass={props.listClass}
                getKey={(item) => item.id}
                renderRow={(item) => <ItemCatalogRow item={item} />}
              />
            </TabsContent>
          </Tabs>
        }
      >
        <UnifiedResults
          query={query()}
          catalogs={{
            conditions: conditions(),
            spells: spells(),
            powers: powers(),
            items: items(),
          }}
          listClass={props.listClass}
        />
      </Show>
    </div>
  )
}

/** One tab's catalog, virtualized — the power list alone is ~560 entries. */
function CatalogTab<T>(props: {
  entries: readonly T[]
  estimateSize: number
  listClass?: string
  getKey: (entry: T) => string
  renderRow: (entry: T) => JSX.Element
}) {
  return (
    <div class="flex min-h-0 flex-1 flex-col gap-1.5">
      <p class="text-2xs text-muted-foreground">{props.entries.length} entradas</p>
      <VirtualList
        items={props.entries}
        getKey={props.getKey}
        estimateSize={props.estimateSize}
        class={props.listClass ?? 'min-h-0 flex-1 pr-1'}
        renderItem={props.renderRow}
      />
    </div>
  )
}

/**
 * Every catalog at once, grouped under a header per catalog, in ONE virtual
 * list — a mid-combat lookup should not make the GM guess which tab a rule
 * lives in.
 */
function UnifiedResults(props: {
  query: string
  catalogs: {
    conditions: readonly Condition[]
    spells: readonly CatalogSpell[]
    powers: readonly CatalogPower[]
    items: readonly CatalogItem[]
  }
  listClass?: string
}) {
  const rows = createMemo(() => catalogSearchRows(props.query, props.catalogs))
  const hits = () => rows().filter((row) => row.kind !== 'header').length

  return (
    <Show
      when={hits() > 0}
      fallback={
        <p class="p-6 text-center text-sm text-muted-foreground">
          Nada encontrado em nenhum catálogo.
        </p>
      }
    >
      <div class="flex min-h-0 flex-1 flex-col gap-1.5">
        <p class="text-2xs text-muted-foreground">
          {hits()} resultado{hits() === 1 ? '' : 's'} em todos os catálogos
        </p>
        <VirtualList
          items={rows()}
          getKey={(row) => row.key}
          estimateSize={96}
          class={props.listClass ?? 'min-h-0 flex-1 pr-1'}
          renderItem={(row) => <ResultRow row={row} />}
        />
      </div>
    </Show>
  )
}

function ResultRow(props: { row: CatalogResultRow }) {
  return (
    <Switch>
      <Match when={props.row.kind === 'header' && props.row}>
        {(row) => (
          <SectionLabel tom="gold" class="pt-1">
            {row().label} · {row().count}
          </SectionLabel>
        )}
      </Match>
      <Match when={props.row.kind === 'condition' && props.row}>
        {(row) => <ConditionRow condition={row().value} />}
      </Match>
      <Match when={props.row.kind === 'spell' && props.row}>
        {(row) => <SpellCatalogRow spell={row().value} />}
      </Match>
      <Match when={props.row.kind === 'power' && props.row}>
        {(row) => <PowerCatalogRow power={row().value} />}
      </Match>
      <Match when={props.row.kind === 'item' && props.row}>
        {(row) => <ItemCatalogRow item={row().value} />}
      </Match>
    </Switch>
  )
}
