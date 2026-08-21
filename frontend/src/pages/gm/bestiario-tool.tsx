import { useQuery } from '@tanstack/solid-query'
import type { Monster } from '@/shared/api/catalog-types'
import { Show, createMemo, createSignal } from 'solid-js'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { createMonsterFilter, filterMonsters } from '@/features/gm-tools/monster-filter'
import { MonsterDetail } from '@/features/gm-tools/monster-detail'
import { MonsterFilters } from '@/features/gm-tools/monster-filters'
import { formatNd, MONSTER_TIPO_LABEL } from '@/features/gm-tools/monster-format'
import { createMediaQuery } from '@/shared/lib/media-query'
import { cn } from '@/shared/lib/utils'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog'
import { Skeleton } from '@/shared/ui/skeleton'
import { VirtualList } from '@/shared/ui/virtual-list'
import { SectionTitle } from '@/shared/ui/section-label'

/**
 * Bestiário — filter and list on the left, the chosen creature's stat block on
 * the right. Below `lg` there is no room for the pane, so tapping a row opens
 * the block as a dialog instead: the same detail, reached the way the width
 * allows.
 */
export function BestiarioTool() {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  // `create*` holding state → born once in the component body (gotcha #17).
  const filter = createMonsterFilter()
  const [pickedId, setPickedId] = createSignal<string | null>(null)
  const [dialogId, setDialogId] = createSignal<string | null>(null)
  const isWide = createMediaQuery('(min-width: 1024px)')

  const all = () => bestiary.data ?? []
  const shown = createMemo(() => filterMonsters(all(), filter.criteria()))
  const picked = () => shown().find((m) => m.id === pickedId()) ?? shown()[0]
  const inDialog = () => shown().find((m) => m.id === dialogId()) ?? null

  const open = (monster: Monster) => {
    setPickedId(monster.id)
    // Only the narrow layout needs the dialog — at lg the pane already shows it.
    if (!isWide()) setDialogId(monster.id)
  }

  return (
    <section class="mesa-palco flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="mesa-bestiario">
      <div class="flex flex-wrap items-baseline gap-x-3">
        <SectionTitle
          id="mesa-bestiario">
          Bestiário
        </SectionTitle>
        <p class="text-xs text-muted-foreground">
          {shown().length} de {all().length}
        </p>
      </div>

      <Show when={!bestiary.isPending} fallback={<BestiarySkeleton />}>
        {/* `min-h-0 flex-1` em TODAS as larguras, não só a partir de `lg`
            (ALE-175): abaixo de `lg` a grade não esticava, a lista parava na
            tampa de 45vh e o que sobrava era faixa morta — 243px medidos em
            768×1024, um quarto da tela. */}
        {/* `@container` e não `lg:` (ALE-172): a decisão aqui é "cabe painel
            lateral ao lado da lista?", que é uma pergunta sobre o espaço que
            ESTA grade recebe, não sobre o tamanho da janela. Com o `lg:` a
            resposta era não-monotônica — medido, um contêiner de 800px dava
            DUAS colunas (janela 1024) e um de 968px dava UMA (janela 1000),
            porque a coluna de ferramentas devolve largura à direita conforme a
            janela encolhe. Crescer o espaço custava uma coluna.
            50rem é o contêiner mais estreito que já mostrava duas colunas, de
            modo que nenhum formato muda de cara: o que muda é só a faixa de
            janela 1000–1023, que passa a ter o painel em vez do diálogo. */}
        <div class="mesa-duas-colunas grid min-h-0 flex-1 gap-4">
          <div class="flex min-h-0 flex-col gap-2">
            <MonsterFilters filter={filter} idPrefix="bestiario" />

            <Show
              when={shown().length > 0}
              fallback={
                <p class="p-4 text-center text-xs text-muted-foreground">
                  Nenhuma criatura casa com os filtros.
                </p>
              }
            >
              {/* A tampa de 45vh SAIU (ALE-175). Ela existia para "80 linhas não
                  enterrarem o resto da ferramenta" — mas abaixo de `lg` o resto
                  da ferramenta é o painel de detalhe, que ali é `hidden`. A
                  tampa protegia conteúdo que não existe naquele formato, e o
                  preço era 243px mortos em 768×1024 com a lista mostrando 459px
                  de 5216 de conteúdo.
                  A lista é o ÚLTIMO elemento da coluna: quem a limita passa a
                  ser a caixa, não um número de altura de viewport. */}
              <VirtualList
                items={shown()}
                getKey={(monster) => monster.id}
                estimateSize={72}
                class="min-h-0 flex-1 rounded-sm border border-grimorio-iron p-1"
                renderItem={(monster) => (
                  <MonsterRow
                    monster={monster}
                    selected={picked()?.id === monster.id}
                    onOpen={() => open(monster)}
                  />
                )}
              />
            </Show>
          </div>

          <section
            aria-label="Criatura escolhida"
            class="mesa-painel"
          >
            <Show
              when={picked()}
              fallback={
                <p class="text-xs text-muted-foreground">Escolha uma criatura na lista.</p>
              }
            >
              {(monster) => <MonsterDetail monster={monster()} />}
            </Show>
          </section>
        </div>
      </Show>

      {/* Narrow only: at lg the pane is already showing this block. */}
      <Dialog open={inDialog() !== null} onOpenChange={(open) => !open && setDialogId(null)}>
        <DialogContent class="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
          <Show when={inDialog()}>
            {(monster) => (
              <>
                <DialogHeader class="sr-only">
                  <DialogTitle>{monster().name}</DialogTitle>
                  <DialogDescription>Estatísticas de {monster().name}</DialogDescription>
                </DialogHeader>
                <MonsterDetail monster={monster()} />
              </>
            )}
          </Show>
        </DialogContent>
      </Dialog>
    </section>
  )
}

function MonsterRow(props: { monster: Monster; selected: boolean; onOpen: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={props.selected}
      onClick={() => props.onOpen()}
      class={cn(
        'w-full rounded-sm border p-2 text-left transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-accent'
          : 'border-grimorio-iron hover:bg-accent',
      )}
    >
      <p class="flex flex-wrap items-baseline gap-x-1.5 text-xs font-semibold">
        {props.monster.name}
        <span class="font-mono text-3xs text-grimorio-gold">
          ND {formatNd(props.monster.nd)}
        </span>
        <span class="text-3xs font-normal text-muted-foreground">
          {MONSTER_TIPO_LABEL[props.monster.tipo]} · {props.monster.size}
        </span>
      </p>
      <p class="font-mono text-3xs text-muted-foreground">
        PV {props.monster.hp} · DEF {props.monster.defesa} · p{props.monster.bookPage}
      </p>
    </button>
  )
}

function BestiarySkeleton() {
  return (
    <div class="mesa-duas-colunas grid gap-4">
      <Skeleton class="h-64 w-full" />
      <Skeleton class="mesa-painel h-64 w-full" />
    </div>
  )
}
