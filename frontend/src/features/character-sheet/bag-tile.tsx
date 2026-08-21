import {
  AlertTriangle,
  FlaskConical,
  Package,
  Shield,
  Shirt,
  Sword,
  Utensils,
  Wand2,
} from 'lucide-solid'
import { type Component, Show } from 'solid-js'
import { Dynamic } from 'solid-js/web'
import type { CharacterItem } from '@/shared/api/api'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { itemOverlayNames } from './item-describe'

type Glyph = Component<{ class?: string; 'aria-hidden'?: boolean | 'true' | 'false' }>

/**
 * Category → tile glyph. Custom items (no catalog) read as generic gear.
 * Returns the component rather than an element so the caller renders it with
 * `<Dynamic>`: a `Show` chain would keep the first glyph it painted.
 *
 * @example glyphFor(espadaLongaRow) === Sword
 */
export function glyphFor(item: CharacterItem): Glyph {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const category = catalog?.category
  if (!category) return Package
  if (category.startsWith('weapon-')) return Sword
  if (category.startsWith('armor-') || category === 'shield') return Shield
  if (category === 'apparel') return catalog.equip === 'wielded' ? Wand2 : Shirt
  if (category === 'consumable') return FlaskConical
  if (category === 'meal') return Utensils
  return Package
}

export type BagTileProps = {
  item: CharacterItem
  proficient: boolean
  onOpen: () => void
}

/**
 * One Mochila cell for a stowed item: category glyph, name, ×qty badge and
 * overlay markers. The whole tile is the tap target that opens the item's
 * action sheet.
 */
export function BagTile(props: BagTileProps) {
  const overlays = () => itemOverlayNames(props.item)

  return (
    <button
      type="button"
      onClick={() => props.onOpen()}
      aria-label={`Abrir ${props.item.name}`}
      class="relative flex min-h-[4.5rem] flex-col items-center justify-center gap-1 rounded-none border border-grimorio-iron bg-grimorio-panel p-2 text-center transition-colors hover:border-grimorio-gold/50"
    >
      <Show when={props.item.quantity > 1}>
        <span class="absolute top-1 right-1 rounded-full bg-accent px-1.5 font-mono text-3xs font-semibold text-grimorio-gold">
          ×{props.item.quantity}
        </span>
      </Show>
      <Show when={!props.proficient && props.item.equipped !== null}>
        <AlertTriangle
          aria-label={`${props.item.name} sem proficiência`}
          class="absolute top-1 left-1 size-3 text-destructive"
        />
      </Show>
      <Dynamic
        component={glyphFor(props.item)}
        aria-hidden="true"
        class="size-5 text-muted-foreground"
      />
      <span class="line-clamp-2 w-full text-2xs leading-tight text-foreground">
        {props.item.name}
      </span>
      <Show when={overlays().length > 0}>
        <span class="line-clamp-1 w-full text-4xs text-muted-foreground">
          {overlays().join(' · ')}
        </span>
      </Show>
    </button>
  )
}
