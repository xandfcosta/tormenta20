import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { classTiles } from './grant-helpers'

/**
 * Primary-class picker — a hue-tiled grid mirroring the race grid, each tile
 * carrying the class's level-1 vitals (PV/PM) so the biggest mechanical choice
 * is comparable at a glance. Single-select; additional multiclass entries fall
 * back to combobox rows in the wizard.
 */
export function ClassTileGrid({
  options,
  selected,
  onSelect,
}: {
  options: string[]
  selected: string
  onSelect: (className: string) => void
}) {
  return (
    <div
      role="listbox"
      aria-label="Classe"
      className="grid grid-cols-2 gap-2 sm:grid-cols-3"
    >
      {classTiles(options).map((tile) => {
        const isSelected = tile.className === selected
        const hue = hueFromName(tile.className)
        return (
          <button
            key={tile.className}
            type="button"
            role="option"
            aria-selected={isSelected}
            onClick={() => onSelect(tile.className)}
            style={
              isSelected ? { outlineColor: `oklch(0.6 0.16 ${hue})` } : undefined
            }
            className={cn(
              'flex items-center gap-2 rounded-lg border border-border p-2 text-left transition-colors',
              isSelected
                ? 'bg-accent outline outline-2 outline-offset-2'
                : 'hover:bg-accent',
            )}
          >
            <CharacterPortrait name={tile.className} size="sm" hue={hue} />
            <div className="min-w-0">
              <p className="line-clamp-2 text-sm font-medium leading-tight">
                {tile.className}
              </p>
              <p className="font-mono text-[10px] text-muted-foreground">
                PV {tile.pvInicial} · PM +{tile.mpPerLevel}/nv
              </p>
            </div>
          </button>
        )
      })}
    </div>
  )
}
