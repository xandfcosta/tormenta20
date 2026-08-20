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
    <section class="flex min-h-0 flex-1 flex-col gap-3" aria-labelledby="mesa-bestiario">
      <div class="flex flex-wrap items-baseline gap-x-3">
        <h2
          id="mesa-bestiario"
          class="font-heading text-lg uppercase tracking-[0.16em] text-grimorio-gold"
        >
          Bestiário
        </h2>
        <p class="text-xs text-muted-foreground">
          {shown().length} de {all().length}
        </p>
      </div>

      <Show when={!bestiary.isPending} fallback={<BestiarySkeleton />}>
        {/* `min-h-0 flex-1` em TODAS as larguras, não só a partir de `lg`
            (ALE-175): abaixo de `lg` a grade não esticava, a lista parava na
            tampa de 45vh e o que sobrava era faixa morta — 243px medidos em
            768×1024, um quarto da tela. */}
        <div class="grid min-h-0 flex-1 gap-4 lg:grid-cols-[minmax(16rem,22rem)_1fr]">
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
                class="min-h-0 flex-1 rounded-md border border-grimorio-iron p-1"
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
            class="hidden lg:block lg:min-h-0 lg:overflow-y-auto lg:rounded-md lg:border lg:border-grimorio-iron lg:p-3"
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
        'w-full rounded-md border p-2 text-left transition-colors',
        props.selected
          ? 'border-grimorio-gold bg-accent'
          : 'border-grimorio-iron hover:bg-accent',
      )}
    >
      <p class="flex flex-wrap items-baseline gap-x-1.5 text-xs font-semibold">
        {props.monster.name}
        <span class="font-mono text-[10px] text-grimorio-gold">
          ND {formatNd(props.monster.nd)}
        </span>
        <span class="text-[10px] font-normal text-muted-foreground">
          {MONSTER_TIPO_LABEL[props.monster.tipo]} · {props.monster.size}
        </span>
      </p>
      <p class="font-mono text-[10px] text-muted-foreground">
        PV {props.monster.hp} · DEF {props.monster.defesa} · p{props.monster.bookPage}
      </p>
    </button>
  )
}

function BestiarySkeleton() {
  return (
    <div class="grid gap-4 lg:flex-1 lg:grid-cols-[minmax(16rem,22rem)_1fr]">
      <Skeleton class="h-64 w-full" />
      <Skeleton class="hidden h-64 w-full lg:block" />
    </div>
  )
}
