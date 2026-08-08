import { type ReactNode, useMemo } from 'react'
import type {
  CatalogItem,
  CatalogSpell,
  Condition,
} from '@tormenta20/t20-data'
import { VirtualList } from '@/shared/ui/virtual-list'
import {
  type CatalogPower,
  type CatalogResultRow,
  catalogSearchRows,
} from './catalog-model'
import {
  ConditionRow,
  ItemCatalogRow,
  PowerRow,
  SpellCatalogRow,
} from './catalog-rows'

/**
 * Unified search view across every GM catalog. When the search box has a query,
 * this replaces the tabbed browser: it filters conditions/magias/poderes/itens
 * together and renders the matches in one virtualized list, grouped by catalog
 * under a section header. Fixes the old behavior where the query only filtered
 * the active tab, so "bola de fogo" on the Condições tab read "Nada encontrado"
 * even though the spell existed (ALE-22).
 */
export function CatalogSearchResults({
  conditions,
  spells,
  powers,
  items,
  query,
  listClassName = 'max-h-[65vh] pr-1',
}: {
  conditions: readonly Condition[]
  spells: readonly CatalogSpell[]
  powers: readonly CatalogPower[]
  items: readonly CatalogItem[]
  query: string
  listClassName?: string
}) {
  const rows = useMemo(
    () => catalogSearchRows(query, { conditions, spells, powers, items }),
    [query, conditions, spells, powers, items],
  )
  const total = rows.reduce((n, r) => (r.kind === 'header' ? n : n + 1), 0)

  if (total === 0) {
    return (
      <p className="p-6 text-center text-sm text-muted-foreground">
        Nada encontrado em nenhum catálogo.
      </p>
    )
  }

  return (
    <div className="space-y-2">
      <p className="text-xs text-muted-foreground">
        {total} resultado{total === 1 ? '' : 's'} em todos os catálogos
      </p>
      <VirtualList
        className={listClassName}
        items={rows}
        estimateSize={96}
        gap={8}
        getKey={(r) => r.key}
        renderItem={renderResultRow}
      />
    </div>
  )
}

function renderResultRow(row: CatalogResultRow): ReactNode {
  switch (row.kind) {
    case 'header':
      return <GroupHeader label={row.label} count={row.count} />
    case 'condition':
      return <ConditionRow condition={row.value} />
    case 'spell':
      return <SpellCatalogRow spell={row.value} />
    case 'power':
      return <PowerRow power={row.value} />
    case 'item':
      return <ItemCatalogRow item={row.value} />
  }
}

function GroupHeader({ label, count }: { label: string; count: number }) {
  return (
    <div className="flex items-baseline gap-2 pt-2 pb-1">
      <h3 className="font-display text-sm font-semibold tracking-wide text-[color:var(--primary)]">
        {label}
      </h3>
      <span className="text-xs text-muted-foreground">{count}</span>
    </div>
  )
}
