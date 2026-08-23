import { Search } from 'lucide-solid'
import { createMemo, createSignal, For, type JSX, Match, Show, Switch } from 'solid-js'
import type { CatalogSpell, Condition } from '@/shared/api/catalog-types'
import type { CatalogItem } from '@/shared/api/item-types'
import { allCatalogItems } from '@/shared/lib/catalog-cache'
import { createElementWidth } from '@/shared/lib/element-size'
import { conditionsList } from '@/shared/lib/rules-catalog-cache'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { Input } from '@/shared/ui/input'
import { SectionLabel } from '@/shared/ui/section-label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { VirtualList } from '@/shared/ui/virtual-list'
import {
  type CatalogHitRow,
  type CatalogPower,
  type CatalogVisualRow,
  catalogColumns,
  catalogPowers,
  catalogSearchRows,
  catalogVisualRows,
  emFileiras,
} from './catalog-model'
import {
  ConditionRow,
  ItemCatalogRow,
  PowerCatalogRow,
  SpellCatalogRow,
} from './catalog-rows'

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

  // A largura vem do PAINEL e não da janela: a gaveta e a ferramenta da Mesa
  // dão espaços diferentes ao mesmo componente, e a trilha lateral do /gm come
  // centenas de pixels da janela sem tirar um só do painel (ALE-172).
  const [caixa, setCaixa] = createSignal<HTMLDivElement>()
  const larguraDaCaixa = createElementWidth(caixa)
  const colunas = () => catalogColumns(larguraDaCaixa())

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
    <div ref={setCaixa} class="flex min-h-0 flex-1 flex-col gap-3">
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
            <TabsList class="@container w-full shrink-0">
              <CatalogTabTrigger value="conditions" label="Condições" />
              <CatalogTabTrigger value="spells" label="Magias" />
              <CatalogTabTrigger value="powers" label="Poderes" />
              <CatalogTabTrigger value="items" label="Itens" />
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
                colunas={colunas()}
                estimateSize={88}
                listClass={props.listClass}
                getKey={(condition) => condition.id}
                renderRow={(condition) => <ConditionRow condition={condition} />}
              />
            </TabsContent>
            <TabsContent value="spells" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={spells()}
                colunas={colunas()}
                estimateSize={140}
                listClass={props.listClass}
                getKey={(spell) => spell.id}
                renderRow={(spell) => <SpellCatalogRow spell={spell} />}
              />
            </TabsContent>
            <TabsContent value="powers" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={powers()}
                colunas={colunas()}
                estimateSize={100}
                listClass={props.listClass}
                getKey={(power) => power.id}
                renderRow={(power) => <PowerCatalogRow power={power} />}
              />
            </TabsContent>
            <TabsContent value="items" class="flex min-h-0 flex-1 flex-col">
              <CatalogTab
                entries={items()}
                colunas={colunas()}
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
          colunas={colunas()}
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

/**
 * Uma aba do catálogo, dividindo a faixa em quatro partes iguais.
 *
 * O `TabsList` do kit nasce `inline-flex w-fit`, então o `flex-1` que o gatilho
 * já tinha não tinha o que dividir: as quatro abas ficavam encolhidas à
 * esquerda enquanto a faixa tinha a largura toda. `w-full` dá a faixa, e o
 * `min-w-0` é obrigatório junto do `flex-1` — sem ele o rótulo mais longo
 * empurra a última aba para FORA da faixa, que foi a medida da ALE-122. O
 * aperto é do CONTÊINER e não da viewport: a mesma tela dá 384px na gaveta e a
 * largura inteira na Mesa (ALE-138).
 */
function CatalogTabTrigger(props: { value: string; label: string }) {
  return (
    <TabsTrigger value={props.value} class="min-w-0 flex-1 px-1 @sm:px-3">
      <span class="truncate">{props.label}</span>
    </TabsTrigger>
  )
}

/** One tab's catalog, virtualized — the power list alone is ~560 entries. */
function CatalogTab<T>(props: {
  entries: readonly T[]
  colunas: number
  estimateSize: number
  listClass?: string
  getKey: (entry: T) => string
  renderRow: (entry: T) => JSX.Element
}) {
  // A lista é virtualizada, então "duas colunas" não é grade de CSS: é agrupar
  // as entradas ANTES de entregá-las, porque quem renderiza uma linha por vez
  // não tem como pôr duas lado a lado (ALE-170).
  const fileiras = createMemo(() => emFileiras(props.entries, props.colunas))

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-1.5">
      <p class="text-2xs text-muted-foreground">{props.entries.length} entradas</p>
      <VirtualList
        items={fileiras()}
        getKey={(fileira) => props.getKey(fileira[0] as T)}
        estimateSize={props.estimateSize}
        class={props.listClass ?? 'min-h-0 flex-1 pr-1'}
        renderItem={(fileira) => (
          <Fileira colunas={props.colunas}>
            <For each={fileira}>{(entry) => props.renderRow(entry)}</For>
          </Fileira>
        )}
      />
    </div>
  )
}

/**
 * Uma fileira de até N resultados lado a lado.
 *
 * Folga só na HORIZONTAL de propósito: na vertical as linhas encostam desde
 * sempre, e mudar isso seria mexer no espaçamento de quatro catálogos sem
 * ninguém ter pedido. Com uma coluna, a fileira é indistinguível da linha
 * antiga.
 *
 * A grade usa `repeat(N, …)` mesmo na fileira curta do fim, para o último
 * cartão nascer da largura dos de cima em vez de esticar sozinho.
 */
function Fileira(props: { colunas: number; children: JSX.Element }) {
  return (
    <div
      class="grid gap-x-1.5"
      style={{ 'grid-template-columns': `repeat(${props.colunas}, minmax(0, 1fr))` }}
    >
      {props.children}
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
  colunas: number
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
  const fileiras = createMemo(() => catalogVisualRows(rows(), props.colunas))

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
          items={fileiras()}
          getKey={(fileira) => fileira.key}
          estimateSize={96}
          class={props.listClass ?? 'min-h-0 flex-1 pr-1'}
          renderItem={(fileira) => <FileiraDeResultados fileira={fileira} colunas={props.colunas} />}
        />
      </div>
    </Show>
  )
}

/**
 * Ou o cabeçalho de um catálogo, que ocupa a fileira toda, ou os N resultados
 * dela lado a lado.
 */
function FileiraDeResultados(props: { fileira: CatalogVisualRow; colunas: number }) {
  return (
    <Switch>
      <Match when={props.fileira.kind === 'header' && props.fileira}>
        {(cabecalho) => (
          <SectionLabel tom="gold" class="pt-1">
            {cabecalho().label} · {cabecalho().count}
          </SectionLabel>
        )}
      </Match>
      <Match when={props.fileira.kind === 'cells' && props.fileira}>
        {(fileira) => (
          <Fileira colunas={props.colunas}>
            <For each={fileira().cells}>{(row) => <ResultRow row={row} />}</For>
          </Fileira>
        )}
      </Match>
    </Switch>
  )
}

function ResultRow(props: { row: CatalogHitRow }) {
  return (
    <Switch>
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
