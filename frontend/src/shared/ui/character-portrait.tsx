import { cn } from '@/shared/lib/utils'

/**
 * Placeholder character portrait. Characters carry no image yet (no field on
 * the model), so we render the initials on a muted panel. `lg` fills its
 * column as a tall 3:4 hero image; `sm` is a roster thumbnail. Swap the inner
 * for an <img> once an imageUrl field lands.
 */
export function CharacterPortrait({
  name,
  size,
  className,
  hue,
}: {
  name: string
  size: 'sm' | 'lg'
  className?: string
  /** When set, tints the tile with a per-character hue gradient (white
   *  initials) so a roster of portraits reads authored, not like grey avatar
   *  fallbacks. Omit for the neutral muted panel. */
  hue?: number
}) {
  const tinted = hue !== undefined
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center overflow-hidden rounded-lg font-semibold',
        tinted ? 'text-white' : 'bg-muted text-muted-foreground',
        size === 'lg' ? 'aspect-[3/4] w-full text-6xl' : 'size-11 text-sm',
        className,
      )}
      style={
        tinted
          ? {
              background: `linear-gradient(150deg, oklch(0.55 0.15 ${hue}) 0%, oklch(0.32 0.09 ${hue}) 100%)`,
            }
          : undefined
      }
      aria-hidden
    >
      {initials(name)}
    </div>
  )
}

function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return parts
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase()
}
