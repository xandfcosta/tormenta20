import { useQuery } from '@tanstack/solid-query'
import type { Monster } from '@/shared/api/catalog-types'
import { Show, createMemo } from 'solid-js'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { VirtualList } from '@/shared/ui/virtual-list'
import { createMonsterFilter, filterMonsters } from './monster-filter'
import { MonsterFilters } from './monster-filters'
import { MONSTER_TIPO_LABEL, formatNd } from './monster-format'
import { settledQuery } from '@/shared/lib/settled-query'

export type MonsterPickerListProps = {
  onPick: (monster: Monster) => void
  /** Distinguishes the field ids when two copies are mounted at once. */
  idPrefix: string
  /** Bounds the scroll area where the list is not the whole surface. */
  listClass?: string
}

/**
 * Filter plus bestiary list, ready to be dropped anywhere. Kept separate from
 * the panel that usually holds it because the in-session encounter builder
 * needs it INLINE: there, the composer is itself a side panel, and a second
 * panel would land on top of the very composition being added to.
 */
export function MonsterPickerList(props: MonsterPickerListProps) {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const filter = createMonsterFilter()
  // `settledQuery` e não `.data ?? []`: a leitura pendente SUSPENDE, o
  // `Suspense` do route match desanexa a cena e a tela pisca ao trocar para
  // esta aba — quarta vez que esta armadilha aparece nesta issue (ALE-122).
  const monsters = () => settledQuery(bestiary) ?? []
  const shown = createMemo(() => filterMonsters(monsters(), filter.criteria()))

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-2">
      <MonsterFilters filter={filter} idPrefix={props.idPrefix} />
      <p class="text-[11px] text-muted-foreground">
        {shown().length} de {monsters().length}
      </p>
      <Show
        when={shown().length > 0}
        fallback={
          <p class="p-4 text-center text-xs text-muted-foreground">
            Nenhuma criatura casa com os filtros.
          </p>
        }
      >
        <VirtualList
          items={shown()}
          getKey={(monster) => monster.id}
          estimateSize={56}
          class={props.listClass ?? 'min-h-0 flex-1'}
          renderItem={(monster) => (
            <button
              type="button"
              onClick={() => props.onPick(monster)}
              class="w-full rounded-sm border border-grimorio-iron p-2 text-left transition-colors hover:bg-accent"
            >
              <p class="flex flex-wrap items-baseline gap-x-1.5 text-xs font-semibold">
                {monster.name}
                <span class="font-mono text-[10px] text-grimorio-gold">
                  ND {formatNd(monster.nd)}
                </span>
              </p>
              <p class="font-mono text-[10px] text-muted-foreground">
                {MONSTER_TIPO_LABEL[monster.tipo]} · PV {monster.hp} · DEF {monster.defesa}
              </p>
            </button>
          )}
        />
      </Show>
    </div>
  )
}
