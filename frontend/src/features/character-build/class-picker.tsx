import { For } from 'solid-js'
import { hueFromName } from '@/shared/lib/hue-from-name'
import { cn } from '@/shared/lib/utils'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { classTiles } from './grant-helpers'

export type ClassTileGridProps = {
  /** Class names the backend offers. */
  options: string[]
  /** Chosen classes, in order — the first is the mechanical primary. */
  value: string[]
  onToggle: (className: string) => void
}

/**
 * Ofício picker — the same grammar as the race grid, so the player learns one
 * way of choosing and uses it in both steps. Each tile carries the class's
 * level-1 vitals, which is what makes the biggest mechanical choice comparable
 * at a glance.
 *
 * Multi-select: the FIRST pick is the primary (it seeds PV and the attribute
 * preset); a second one is multiclasse, which the step confirms before adding.
 */
export function ClassTileGrid(props: ClassTileGridProps) {
  return (
    <div
      role="listbox"
      aria-label="Classe"
      aria-multiselectable="true"
      class="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-2 xl:grid-cols-3"
    >
      <For each={classTiles(props.options)}>
        {(tile) => {
          const hue = () => hueFromName(tile.className)
          const selected = () => props.value.includes(tile.className)
          const isPrimary = () => props.value[0] === tile.className
          return (
            <button
              type="button"
              role="option"
              aria-selected={selected()}
              onClick={() => props.onToggle(tile.className)}
              style={selected() ? { 'outline-color': `oklch(0.6 0.16 ${hue()})` } : undefined}
              class={cn(
                'flex items-center gap-2 rounded-md border border-grimorio-iron p-2 text-left transition-colors',
                selected()
                  ? 'bg-accent outline outline-2 outline-offset-2'
                  : 'hover:bg-accent',
              )}
            >
              <CharacterPortrait name={tile.className} size="sm" hue={hue()} />
              <div class="min-w-0">
                <p class="line-clamp-2 text-sm font-medium leading-tight">
                  {tile.className}
                  <span class="sr-only">{isPrimary() ? ' · classe principal' : ''}</span>
                </p>
                <p class="font-mono text-[10px] text-muted-foreground">
                  PV {tile.pvInicial} · PM +{tile.mpPerLevel}/nv
                </p>
              </div>
            </button>
          )
        }}
      </For>
    </div>
  )
}
