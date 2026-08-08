import { useMemo, useState } from 'react'
import { allCatalogItems } from '@/shared/lib/catalog-cache'
import { conditionsList } from '@/shared/lib/rules-catalog-cache'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { Input } from '@/shared/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import {
  catalogPowers,
  conditionSearch,
  itemSearch,
  powerSearch,
  spellSearch,
} from './catalog-model'
import { CatalogSearchResults } from './catalog-search-results'
import { CatalogTab } from './catalog-tab'
import {
  ConditionRow,
  ItemCatalogRow,
  PowerRow,
  SpellCatalogRow,
} from './catalog-rows'

/**
 * Tabbed catalog browser (condições / magias / poderes / itens) over one shared
 * search box. Owns its own query state so it drops into both the standalone
 * `/gm/catalogs` page and the in-session GM drawer unchanged. Each tab is a
 * virtualized list so the big catalogs (~200 spells, ~560 powers) stay fast.
 *
 * @example <CatalogBrowser listClassName="max-h-[70vh]" />
 */
export function CatalogBrowser({
  listClassName,
}: {
  listClassName?: string
}) {
  const [query, setQuery] = useState('')
  // Item catalog comes from the fetched cache (primed by the root loader), so
  // sort inside the component rather than a module const (which would be empty).
  const itemList = useMemo(
    () =>
      [...allCatalogItems()].sort((a, b) => a.name.localeCompare(b.name, 'pt-BR')),
    [],
  )
  // Same reason as itemList — the power catalog is primed by the loader gate.
  const powerList = useMemo(() => catalogPowers(), [])
  const spellList = useMemo(
    () =>
      Object.values(spellCatalog()).sort(
        (a, b) => a.circle - b.circle || a.name.localeCompare(b.name, 'pt-BR'),
      ),
    [],
  )
  const conditionList = useMemo(
    () =>
      [...conditionsList()].sort((a, b) =>
        a.name.localeCompare(b.name, 'pt-BR'),
      ),
    [],
  )

  // A live query searches every catalog at once (unified results); an empty box
  // falls back to tab-by-tab browsing. This makes the "todos os catálogos"
  // placeholder honest — before, the filter only hit the active tab (ALE-22).
  const searching = query.trim().length > 0

  return (
    <div className="flex min-h-0 flex-col gap-4">
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Buscar em todos os catálogos…"
      />

      {searching ? (
        <CatalogSearchResults
          conditions={conditionList}
          spells={spellList}
          powers={powerList}
          items={itemList}
          query={query}
          listClassName={listClassName}
        />
      ) : (
      <Tabs defaultValue="conditions" className="flex min-h-0 flex-col">
        <TabsList className="max-w-full self-start overflow-x-auto [&>button]:shrink-0">
          <TabsTrigger value="conditions">Condições</TabsTrigger>
          <TabsTrigger value="spells">Magias</TabsTrigger>
          <TabsTrigger value="powers">Poderes</TabsTrigger>
          <TabsTrigger value="items">Itens</TabsTrigger>
        </TabsList>

        <TabsContent value="conditions">
          <CatalogTab
            items={conditionList}
            query={query}
            searchText={conditionSearch}
            estimateSize={80}
            listClassName={listClassName}
            getKey={(c) => c.id}
            renderRow={(c) => <ConditionRow condition={c} />}
          />
        </TabsContent>
        <TabsContent value="spells">
          <CatalogTab
            items={spellList}
            query={query}
            searchText={spellSearch}
            estimateSize={132}
            listClassName={listClassName}
            getKey={(s) => s.id}
            renderRow={(s) => <SpellCatalogRow spell={s} />}
          />
        </TabsContent>
        <TabsContent value="powers">
          <CatalogTab
            items={powerList}
            query={query}
            searchText={powerSearch}
            estimateSize={96}
            listClassName={listClassName}
            getKey={(p) => p.id}
            renderRow={(p) => <PowerRow power={p} />}
          />
        </TabsContent>
        <TabsContent value="items">
          <CatalogTab
            items={itemList}
            query={query}
            searchText={itemSearch}
            estimateSize={76}
            listClassName={listClassName}
            getKey={(i) => i.id}
            renderRow={(i) => <ItemCatalogRow item={i} />}
          />
        </TabsContent>
      </Tabs>
      )}
    </div>
  )
}
