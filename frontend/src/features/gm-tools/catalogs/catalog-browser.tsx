import { useState } from 'react'
import {
  CATALOG_ITEMS,
  CONDITIONS,
  SPELL_CATALOG,
  type CatalogItem,
  type CatalogSpell,
  type Condition,
} from '@tormenta20/t20-data'
import { Input } from '@/shared/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { CATALOG_POWERS, type CatalogPower } from './catalog-model'
import { CatalogTab } from './catalog-tab'
import {
  ConditionRow,
  ItemCatalogRow,
  PowerRow,
  SpellCatalogRow,
} from './catalog-rows'

// Static, sorted once. SPELL_CATALOG / CONDITIONS are id-keyed records; the
// item list is already an array. One shared search box filters the active tab.
const CONDITION_LIST = Object.values(CONDITIONS).sort((a, b) =>
  a.name.localeCompare(b.name, 'pt-BR'),
)
const SPELL_LIST = Object.values(SPELL_CATALOG).sort(
  (a, b) => a.circle - b.circle || a.name.localeCompare(b.name, 'pt-BR'),
)
const ITEM_LIST = [...CATALOG_ITEMS].sort((a, b) =>
  a.name.localeCompare(b.name, 'pt-BR'),
)

// Stable module-level search extractors (CatalogTab memoizes on their identity).
const conditionSearch = (c: Condition) => [c.name, c.description, ...c.tags]
const spellSearch = (s: CatalogSpell) => [s.name, s.baseEffect]
const powerSearch = (p: CatalogPower) => [p.name, p.source, p.description]
const itemSearch = (i: CatalogItem) => [i.name, i.category]

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

  return (
    <div className="flex min-h-0 flex-col gap-4">
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Buscar em todos os catálogos…"
      />

      <Tabs defaultValue="conditions" className="flex min-h-0 flex-col">
        <TabsList className="max-w-full self-start overflow-x-auto [&>button]:shrink-0">
          <TabsTrigger value="conditions">Condições</TabsTrigger>
          <TabsTrigger value="spells">Magias</TabsTrigger>
          <TabsTrigger value="powers">Poderes</TabsTrigger>
          <TabsTrigger value="items">Itens</TabsTrigger>
        </TabsList>

        <TabsContent value="conditions">
          <CatalogTab
            items={CONDITION_LIST}
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
            items={SPELL_LIST}
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
            items={CATALOG_POWERS}
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
            items={ITEM_LIST}
            query={query}
            searchText={itemSearch}
            estimateSize={76}
            listClassName={listClassName}
            getKey={(i) => i.id}
            renderRow={(i) => <ItemCatalogRow item={i} />}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}
