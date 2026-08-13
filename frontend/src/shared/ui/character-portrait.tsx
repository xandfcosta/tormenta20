import { initials } from '@/shared/lib/initials'
import { cn } from '@/shared/lib/utils'

export type CharacterPortraitProps = {
  name: string
  size: 'sm' | 'lg'
  class?: string
  /**
   * When set, tints the tile with a per-character hue gradient (white
   * initials) so a roster reads authored, not like grey avatar fallbacks.
   * Omit for the neutral muted panel.
   */
  hue?: number
}

/**
 * Placeholder character portrait. Characters carry no image yet (no field on
 * the model), so we render the initials on a muted panel. `lg` fills its
 * column as a tall 3:4 hero image; `sm` is a roster thumbnail. Swap the inner
 * for an `<img>` once an imageUrl field lands.
 *
 * @example <CharacterPortrait name="Tanque Placas" size="sm" hue={210} />
 */
export function CharacterPortrait(props: CharacterPortraitProps) {
  const tinted = () => props.hue !== undefined
  return (
    <div
      class={cn(
        'flex shrink-0 items-center justify-center overflow-hidden rounded-lg font-semibold',
        tinted() ? 'text-white' : 'bg-muted text-muted-foreground',
        props.size === 'lg' ? 'aspect-[3/4] w-full text-6xl' : 'size-11 text-sm',
        props.class,
      )}
      style={
        tinted()
          ? {
              background: `linear-gradient(150deg, oklch(0.55 0.15 ${props.hue}) 0%, oklch(0.32 0.09 ${props.hue}) 100%)`,
            }
          : undefined
      }
      // Explicit "true" — bare `aria-hidden` renders as `aria-hidden=""` in
      // Solid, which does not hide the node (the name is on the row anyway).
      aria-hidden="true"
    >
      {initials(props.name)}
    </div>
  )
}

