import { allMonsters } from '@/shared/lib/bestiary-cache'
import type { Monster } from '@/shared/api/catalog-types'
import { Show, createMemo } from 'solid-js'
import { VirtualList } from '@/shared/ui/virtual-list'
import { createMonsterFilter, filterMonsters } from './monster-filter'
import { MonsterFilters } from './monster-filters'
import { MONSTER_TIPO_LABEL, formatNd } from './monster-format'

export type MonsterPickerListProps = {
  onPick: (monster: Monster) => void
  /** Distinguishes the field ids when two copies are mounted at once. */
  idPrefix: string
  /** Bounds the scroll area where the list is not the whole surface. */
  listClass?: string
  /**
   * O VERBO da linha, dito antes do nome da criatura para quem ouve. Ele muda
   * por superfície: no bestiário da sessão a linha ABRE (ALE-208), no Montar
   * encontro ela ADICIONA ao rascunho — um nome só para os dois mentiria num
   * deles.
   *
   * É um prefixo `sr-only` DENTRO do botão, e não um `aria-label`, porque o
   * `aria-label` SUBSTITUI o nome inteiro: a linha perdia o ND, o tipo, o PV e
   * a Defesa que ela anuncia hoje, e o `gm-listas.spec.ts` — que procura as
   * linhas por `/ND /` — foi quem acusou. O prefixo soma; o rótulo trocava.
   *
   * Sem ele a linha se chama só pela criatura, que é o que a Mesa do Mestre quer.
   */
  itemVerbo?: string
}

/**
 * Filter plus bestiary list, ready to be dropped anywhere. Kept separate from
 * the panel that usually holds it because the in-session encounter builder
 * needs it INLINE: there, the composer is itself a side panel, and a second
 * panel would land on top of the very composition being added to.
 */
export function MonsterPickerList(props: MonsterPickerListProps) {
  const filter = createMonsterFilter()
  // Acessor SÍNCRONO e não `useQuery`: aqui `settledQuery` não bastava. Quem
  // suspende é o RECURSO NASCER dentro de um dono reativo novo, e o portal do
  // Kobalte cria um a cada abertura — medido na ALE-199, o desanexo acontecia
  // até com o cache já preparado. O bestiário é preparado no `ensureCatalogs`.
  const monsters = () => allMonsters()
  const shown = createMemo(() => filterMonsters(monsters(), filter.criteria()))

  return (
    <div class="flex min-h-0 flex-1 flex-col gap-2">
      <MonsterFilters filter={filter} idPrefix={props.idPrefix} />
      <p class="text-2xs text-muted-foreground">
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
              <Show when={props.itemVerbo}>
                <span class="sr-only">{props.itemVerbo} </span>
              </Show>
              <p class="flex flex-wrap items-baseline gap-x-1.5 text-xs font-semibold">
                {monster.name}
                <span class="font-mono text-3xs text-grimorio-gold">
                  ND {formatNd(monster.nd)}
                </span>
              </p>
              <p class="font-mono text-3xs text-muted-foreground">
                {MONSTER_TIPO_LABEL[monster.tipo]} · PV {monster.hp} · DEF {monster.defesa}
              </p>
            </button>
          )}
        />
      </Show>
    </div>
  )
}
