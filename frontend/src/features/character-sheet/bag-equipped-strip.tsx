import { X } from 'lucide-solid'
import { For, Index, type ParentProps, Show } from 'solid-js'
import type { CharacterItem } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import type { BagPartition } from './bag-slots'
import { equipBonuses } from './equip-bonuses'
import { itemOverlayNames } from './item-describe'

export type BagEquippedStripProps = {
  partition: BagPartition
  onOpen: (item: CharacterItem) => void
  onUnequip: (item: CharacterItem) => void
}

/**
 * The bag's paper-doll strip — the two capacity pools the rules actually track
 * (there are no body slots in T20): Mãos (≤2 hand-slots; a `wielded2` weapon
 * takes both) and Vestidos (≤4). Filled cards open the item's action sheet;
 * the ✕ is a quick unequip that skips the sheet.
 */
export function BagEquippedStrip(props: BagEquippedStripProps) {
  const vestedSlots = () => [0, 1, 2, 3].map((i) => props.partition.vested[i])

  return (
    // @container e não `lg:`: esta tira vive tanto na ficha larga do jogador
    // quanto numa COLUNA de 518px (o painel do combatente do mestre). Por
    // viewport, numa janela de 1920 ela pegava o layout lado a lado dentro dos
    // 518px e os seis cartões ficavam com 72px cada — nome truncado em "A…" e o
    // crachá de bônus saindo por cima do cartão vizinho (ALE-148). Mesma
    // armadilha da ALE-122 e da lista de perícias na ALE-145.
    <div class="@container grid gap-3 @[44rem]:grid-cols-[1fr_2fr]">
      <PoolBox title="Mãos" count={props.partition.handsUsed} max={2}>
        <Show
          when={props.partition.twoHand}
          fallback={
            <>
              <EquippedCard
                label="Mão principal"
                item={props.partition.wielded[0]}
                onOpen={props.onOpen}
                onUnequip={props.onUnequip}
              />
              <EquippedCard
                label="Mão secundária"
                item={props.partition.wielded[1]}
                onOpen={props.onOpen}
                onUnequip={props.onUnequip}
              />
            </>
          }
        >
          {(twoHand) => (
            <EquippedCard
              label="Duas mãos"
              item={twoHand()}
              onOpen={props.onOpen}
              onUnequip={props.onUnequip}
              wide
            />
          )}
        </Show>
      </PoolBox>

      <PoolBox title="Vestidos" count={props.partition.vested.length} max={4} columns={4}>
        {/* Index, not For: the four Vestido slots are positions, and empty ones
            are all `undefined` — For keys by reference and cannot tell them
            apart. */}
        <Index each={vestedSlots()}>
          {(item) => (
            <EquippedCard
              label="Vestido"
              item={item()}
              onOpen={props.onOpen}
              onUnequip={props.onUnequip}
            />
          )}
        </Index>
      </PoolBox>
    </div>
  )
}

/** Titled capacity pool: x/max counter over a slot grid (2 cols on phone). */
function PoolBox(
  props: ParentProps<{ title: string; count: number; max: number; columns?: 2 | 4 }>,
) {
  return (
    // Contêiner PRÓPRIO: quantas colunas cabem depende da largura DESTE box,
    // que é 1/3 ou 2/3 da tira quando ela está lado a lado.
    <div class="@container space-y-1.5">
      <div class="flex items-baseline justify-between">
        <h3 class="font-heading text-[10px] font-bold uppercase tracking-widest text-grimorio-gold">
          {props.title}
        </h3>
        <span
          class={cn(
            'font-mono text-xs',
            props.count >= props.max ? 'text-foreground' : 'text-muted-foreground',
          )}
        >
          {props.count}/{props.max}
        </span>
      </div>
      <div class={cn('grid grid-cols-2 gap-2', props.columns === 4 && '@[28rem]:grid-cols-4')}>
        {props.children}
      </div>
    </div>
  )
}

function EquippedCard(props: {
  label: string
  item: CharacterItem | undefined
  onOpen: (item: CharacterItem) => void
  onUnequip: (item: CharacterItem) => void
  wide?: boolean
}) {
  return (
    <Show when={props.item} fallback={<EmptySlot label={props.label} wide={props.wide} />}>
      {(item) => (
        <div
          class={cn(
            'relative min-h-[3.75rem] rounded-none border border-grimorio-gold/40 bg-grimorio-panel-raised px-2.5 py-2 text-left',
            props.wide && 'col-span-2',
          )}
        >
          <button
            type="button"
            onClick={() => props.onOpen(item())}
            aria-label={`Abrir ${item().name}`}
            class="block w-[calc(100%-1.5rem)] text-left"
          >
            <span class="block text-[9px] uppercase tracking-widest text-muted-foreground">
              {props.label}
            </span>
            <span class="block truncate text-sm font-semibold text-grimorio-gold" title={item().name}>
              {item().name}
            </span>
            <EquippedChips item={item()} />
          </button>
          <button
            type="button"
            onClick={() => props.onUnequip(item())}
            aria-label={`Desequipar ${item().name}`}
            class="absolute top-1.5 right-1.5 inline-flex size-5 items-center justify-center rounded-full text-foreground transition-colors hover:bg-destructive/20 hover:text-destructive"
          >
            <X aria-hidden="true" class="size-3" />
          </button>
        </div>
      )}
    </Show>
  )
}

/** What the slot grants: overlays first (Reforçada, Aço-rubi), then bonuses. */
function EquippedChips(props: { item: CharacterItem }) {
  const chips = () => [...itemOverlayNames(props.item), ...equipBonuses(props.item)]
  return (
    <Show when={chips().length > 0}>
      <span class="mt-1 flex flex-wrap gap-1">
        <For each={chips()}>
          {(chip) => (
            // `max-w-full truncate`: um chip é uma caixa indivisível, então
            // `flex-wrap` não o quebra — "Perícia Intimidação +1" simplesmente
            // saía do cartão e era desenhado por cima do vizinho (ALE-148). O
            // `title` guarda o texto inteiro para quem passar o mouse.
            <span
              title={chip}
              class="max-w-full truncate rounded-full border border-border bg-muted px-1.5 py-0.5 text-[10px] text-foreground"
            >
              {chip}
            </span>
          )}
        </For>
      </span>
    </Show>
  )
}

function EmptySlot(props: { label: string; wide?: boolean }) {
  return (
    <div
      class={cn(
        'flex min-h-[3.75rem] flex-col justify-center rounded-none border border-dashed border-grimorio-iron bg-grimorio-panel px-2.5 py-2',
        props.wide && 'col-span-2',
      )}
    >
      <span class="text-[9px] uppercase tracking-widest text-muted-foreground">{props.label}</span>
      <span class="text-xs text-muted-foreground">vazio</span>
    </div>
  )
}
