import {
  AlertTriangle,
  FlaskConical,
  Package,
  Shield,
  Shirt,
  Sword,
  Utensils,
  Wand2,
} from 'lucide-react'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import type { CharacterItem } from '@/shared/api/api'
import { dimText, hoverRow } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { itemOverlayNames } from './item-describe'

/** Category → tile glyph. Custom items (no catalog) read as generic gear. */
function TileGlyph({ item, className }: { item: CharacterItem; className?: string }) {
  const catalog = item.catalogId ? getCatalogItem(item.catalogId) : undefined
  const category = catalog?.category
  if (!category) return <Package className={className} aria-hidden />
  if (category.startsWith('weapon-')) return <Sword className={className} aria-hidden />
  if (category.startsWith('armor-') || category === 'shield')
    return <Shield className={className} aria-hidden />
  if (category === 'apparel')
    return catalog.equip === 'wielded' ? (
      <Wand2 className={className} aria-hidden />
    ) : (
      <Shirt className={className} aria-hidden />
    )
  if (category === 'consumable') return <FlaskConical className={className} aria-hidden />
  if (category === 'meal') return <Utensils className={className} aria-hidden />
  return <Package className={className} aria-hidden />
}

/**
 * One Mochila tile — the game-bag cell for a stowed item: category glyph,
 * name, ×qty badge and overlay markers. The whole tile is the tap target
 * that opens the item's action sheet.
 */
export function BagTile({
  item,
  proficient,
  onOpen,
}: {
  item: CharacterItem
  proficient: boolean
  onOpen: () => void
}) {
  const overlays = itemOverlayNames(item)
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label={`Abrir ${item.name}`}
      className={cn(
        'relative flex min-h-[4.5rem] flex-col items-center justify-center gap-1 rounded-lg border border-border bg-muted/40 p-2 text-center transition-colors',
        hoverRow,
      )}
    >
      {item.quantity > 1 && (
        <span className="absolute right-1 top-1 rounded-full bg-primary/15 px-1.5 font-mono text-[10px] font-semibold text-primary">
          ×{item.quantity}
        </span>
      )}
      {!proficient && item.equipped !== null && (
        <AlertTriangle className="absolute left-1 top-1 size-3 text-red-700 dark:text-red-400" />
      )}
      <TileGlyph item={item} className={cn('size-5', dimText)} />
      <span className="line-clamp-2 w-full text-[11px] leading-tight text-foreground">
        {item.name}
      </span>
      {overlays.length > 0 && (
        <span className={cn('line-clamp-1 w-full text-[9px]', dimText)}>
          {overlays.join(' · ')}
        </span>
      )}
    </button>
  )
}
