import { useQuery } from '@tanstack/solid-query'
import type { Monster } from '@tormenta20/t20-data'
import { type JSX, Show, createMemo } from 'solid-js'
import { bestiaryCatalogQueryOptions } from '@/entities/catalog/queries'
import { SidePanel } from '@/shared/ui/side-panel'
import { VirtualList } from '@/shared/ui/virtual-list'
import { createMonsterFilter, filterMonsters } from './monster-filter'
import { MonsterFilters } from './monster-filters'
import { MONSTER_TIPO_LABEL, formatNd } from './monster-format'

export type MonsterPickerPanelProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  /** Live context pinned above the list (the tracker's peek, in a session). */
  header?: JSX.Element
  onPick: (monster: Monster) => void
  /** Whether picking closes the panel. Adding several creatures in a row is the
   *  normal case, so the default keeps it open. */
  closeOnPick?: boolean
}

/**
 * Pick a creature out of the bestiary, in a side panel. Shared by the Mesa's
 * encounter builder and by the in-session "adicionar monstro", so the filter a
 * GM learned at the table is the same one they use mid-combat.
 */
export function MonsterPickerPanel(props: MonsterPickerPanelProps) {
  const bestiary = useQuery(() => bestiaryCatalogQueryOptions)
  const filter = createMonsterFilter()
  const shown = createMemo(() => filterMonsters(bestiary.data ?? [], filter.criteria()))

  const pick = (monster: Monster) => {
    props.onPick(monster)
    if (props.closeOnPick) props.onOpenChange(false)
  }

  return (
    <SidePanel
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={props.title}
      description={props.description}
      header={props.header}
    >
      <div class="flex min-h-0 flex-1 flex-col gap-2">
        <MonsterFilters filter={filter} idPrefix="picker" />
        <p class="text-[11px] text-muted-foreground">
          {shown().length} de {(bestiary.data ?? []).length}
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
            class="min-h-0 flex-1"
            renderItem={(monster) => (
              <button
                type="button"
                onClick={() => pick(monster)}
                class="w-full rounded-md border border-grimorio-iron p-2 text-left transition-colors hover:bg-accent"
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
    </SidePanel>
  )
}
