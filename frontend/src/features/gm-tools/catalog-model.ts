import type { CatalogSpell, Condition } from '@/shared/api/catalog-types'
import type { CatalogItem } from '@/shared/api/item-types'
import {
  classPowerCatalog,
  generalPowerCatalog,
  grantedPowers as grantedPowerCatalog,
} from '@/shared/lib/abilities-cache'
import { normalizeText } from '@/shared/lib/normalize-text'

/**
 * One power for the GM's lookup. The book scatters powers across several
 * catalogs (class abilities, general/combat feats, granted divine powers); the
 * GM just wants ONE searchable "Poderes" list, so they are flattened to a
 * common shape tagged by where they came from. Divine powers are left out —
 * their data carries a book page and no rules text to check.
 */
export type CatalogPower = {
  id: string
  name: string
  source: string
  description: string
}

/**
 * Every searchable power, sorted by name. A function and NOT a module-level
 * const: the abilities catalog is fetched and primed by the loader gate, so
 * reading it at import time — before priming — would freeze an empty list
 * (gotcha #13 of the port).
 */
export function catalogPowers(): CatalogPower[] {
  const fromClasses = classPowerCatalog().map((power) => ({
    id: power.id,
    name: power.name,
    source: power.className,
    description: power.description,
  }))
  const fromGeneral = generalPowerCatalog().map((power) => ({
    id: `general.${power.id}`,
    name: power.name,
    source: `Geral · ${power.kind}`,
    description: power.description,
  }))
  const fromGods = grantedPowerCatalog().map((power) => ({
    id: `granted.${power.id}`,
    name: power.name,
    source: `Concedido · ${power.deuses.join(', ')}`,
    description: power.effect,
  }))
  return [...fromClasses, ...fromGeneral, ...fromGods].sort((a, b) =>
    a.name.localeCompare(b.name, 'pt-BR'),
  )
}

/**
 * True when EVERY whitespace-separated term appears in one of the fields.
 * Terms are ANDed, so "luz cur" narrows to entries carrying both. An empty
 * query matches everything.
 *
 * Deliberately not `shared/lib/fuzzy-filter`'s `matchesQuery`: that one is
 * typo-tolerant ranking for picking ONE thing out of a list. Here the GM is
 * narrowing a reference by words they know, and fuzzy ranking would drag in
 * near-misses that make a rules lookup feel wrong.
 *
 * @example matchesAllTerms(['Bola de Fogo', 'dano de fogo'], 'bola fogo') // true
 */
export function matchesAllTerms(fields: readonly string[], query: string): boolean {
  const needle = normalizeText(query)
  if (!needle) return true
  const haystack = normalizeText(fields.join(' '))
  return needle.split(/\s+/).every((term) => haystack.includes(term))
}

// What each catalog matches on. Module-level so the browse tabs and the unified
// search agree by construction instead of by copy.
export const conditionSearch = (c: Condition): string[] => [c.name, c.description, ...c.tags]
export const spellSearch = (s: CatalogSpell): string[] => [s.name, s.baseEffect]
export const powerSearch = (p: CatalogPower): string[] => [p.name, p.source, p.description]
export const itemSearch = (i: CatalogItem): string[] => [i.name, i.category]

/** A flat row of the unified search: a per-catalog header followed by its hits,
 *  so ONE virtualized list can render results of four different shapes. */
export type CatalogResultRow =
  | { kind: 'header'; key: string; label: string; count: number }
  | { kind: 'condition'; key: string; value: Condition }
  | { kind: 'spell'; key: string; value: CatalogSpell }
  | { kind: 'power'; key: string; value: CatalogPower }
  | { kind: 'item'; key: string; value: CatalogItem }

/**
 * Uma linha VISUAL da busca: ou um cabeçalho, que ocupa a largura toda, ou uma
 * fileira de N resultados lado a lado.
 *
 * Existe porque a lista é VIRTUALIZADA (ALE-170): ela renderiza uma linha de
 * dados por vez, então "duas ou três colunas" não é uma grade de CSS — é
 * agrupar os dados antes de entregá-los.
 */
export type CatalogVisualRow =
  | { kind: 'header'; key: string; label: string; count: number }
  | { kind: 'cells'; key: string; cells: CatalogHitRow[] }

/**
 * Agrupa as linhas em fileiras de `columns` colunas.
 *
 * Duas regras, e as duas importam. Um cabeçalho fica SOZINHO na fileira dele —
 * ele nomeia o grupo inteiro e não pode dividir espaço com um resultado. E o
 * agrupamento REINICIA a cada cabeçalho: sem isso a última condição dividiria
 * a fileira com a primeira magia, e o mestre leria duas coisas de catálogos
 * diferentes lado a lado como se fossem irmãs.
 *
 * @example catalogVisualRows(linhas, 2) // [header, cells×2, cells×1, header, …]
 */
export function catalogVisualRows(
  rows: readonly CatalogResultRow[],
  columns: number,
): CatalogVisualRow[] {
  const fileiras: CatalogVisualRow[] = []
  let grupo: CatalogHitRow[] = []

  const fecha = () => {
    for (const cells of emFileiras(grupo, columns)) {
      fileiras.push({ kind: 'cells', key: `cells.${cells[0]?.key}`, cells })
    }
    grupo = []
  }

  for (const row of rows) {
    if (row.kind !== 'header') {
      grupo.push(row)
      continue
    }
    fecha()
    fileiras.push(row)
  }
  fecha()
  return fileiras
}

/**
 * Fatia uma lista em fileiras de `colunas` itens, com a última fileira curta.
 *
 * @example emFileiras(['a', 'b', 'c'], 2) // [['a', 'b'], ['c']]
 */
export function emFileiras<T>(itens: readonly T[], colunas: number): T[][] {
  const largura = Math.max(1, Math.trunc(colunas))
  const fileiras: T[][] = []
  for (let i = 0; i < itens.length; i += largura) {
    fileiras.push(itens.slice(i, i + largura))
  }
  return fileiras
}

/** Abaixo disto a prosa de uma magia vira uma fita de duas palavras por linha. */
const LARGURA_MINIMA_DE_CARTAO = 360

/** Acima de três a linha de texto fica curta demais para descrição de regra. */
const MAXIMO_DE_COLUNAS = 3

/**
 * Quantas colunas cabem num painel de `largura` px.
 *
 * O teto de três não é estética: é a MEDIDA DE LEITURA. Num painel de 1920 uma
 * coluna só dava ~122 caracteres por linha — mais que o dobro do confortável, e
 * o olho perde a linha na volta. Três colunas põem isso em ~70, que é a faixa
 * que a tipografia recomenda, e de quebra cabe três vezes mais condição na
 * tela (ALE-170).
 *
 * Largura 0 significa "ainda não medi" e cai em uma coluna, que é o arranjo
 * que sempre serve.
 *
 * @example catalogColumns(1200) // 3
 */
export function catalogColumns(largura: number): number {
  const cabem = Math.floor(largura / LARGURA_MINIMA_DE_CARTAO)
  return Math.min(MAXIMO_DE_COLUNAS, Math.max(1, cabem))
}

/** Uma linha da busca que É um resultado — tudo menos o cabeçalho de grupo. */
export type CatalogHitRow = Exclude<CatalogResultRow, { kind: 'header' }>

export type SearchableCatalogs = {
  conditions: readonly Condition[]
  spells: readonly CatalogSpell[]
  powers: readonly CatalogPower[]
  items: readonly CatalogItem[]
}

/**
 * Filters every catalog by one query and flattens the hits into a single
 * grouped row list. Catalogs with no hits are omitted entirely — a header over
 * nothing is noise in a mid-combat lookup.
 *
 * @example catalogSearchRows('bola de fogo', catalogs) // [header Magias, spell…]
 */
export function catalogSearchRows(
  query: string,
  catalogs: SearchableCatalogs,
): CatalogResultRow[] {
  const rows: CatalogResultRow[] = []
  const hits = <T>(list: readonly T[], fields: (entry: T) => string[]) =>
    list.filter((entry) => matchesAllTerms(fields(entry), query))

  pushGroup(rows, 'Condições', hits(catalogs.conditions, conditionSearch), (c) => ({
    kind: 'condition',
    key: `condition.${c.id}`,
    value: c,
  }))
  pushGroup(rows, 'Magias', hits(catalogs.spells, spellSearch), (s) => ({
    kind: 'spell',
    key: `spell.${s.id}`,
    value: s,
  }))
  pushGroup(rows, 'Poderes', hits(catalogs.powers, powerSearch), (p) => ({
    kind: 'power',
    key: `power.${p.id}`,
    value: p,
  }))
  pushGroup(rows, 'Itens', hits(catalogs.items, itemSearch), (i) => ({
    kind: 'item',
    key: `item.${i.id}`,
    value: i,
  }))
  return rows
}

function pushGroup<T>(
  rows: CatalogResultRow[],
  label: string,
  matches: readonly T[],
  toRow: (entry: T) => CatalogResultRow,
): void {
  if (matches.length === 0) return
  rows.push({ kind: 'header', key: `header.${label}`, label, count: matches.length })
  for (const match of matches) rows.push(toRow(match))
}
