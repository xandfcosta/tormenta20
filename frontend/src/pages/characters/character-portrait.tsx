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
}: {
  name: string
  size: 'sm' | 'lg'
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted font-semibold text-muted-foreground',
        size === 'lg' ? 'aspect-[3/4] w-full text-6xl' : 'size-11 text-sm',
        className,
      )}
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
