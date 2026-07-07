import { Link } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import type { NavDestination } from './nav-destinations'

/**
 * Fixed bottom nav for phone viewports. Only shows `primary` tier
 * destinations (max 4) so each tap target has ≥ 25% of the screen
 * width. Uses `env(safe-area-inset-bottom)` so we don't collide with
 * the iOS home indicator, and `dvh`-driven parent so Android's
 * keyboard reflow lifts the nav out of the way rather than covering
 * inputs.
 *
 * Hidden on md+ — desktop reads the same routes from the top nav.
 */
type BottomNavProps = {
  destinations: readonly NavDestination[]
  className?: string
}

function BottomNav({ destinations, className }: BottomNavProps) {
  const primary = destinations.filter((d) => d.tier === 'primary').slice(0, 4)
  if (primary.length === 0) return null

  return (
    <nav
      data-slot="bottom-nav"
      aria-label="Navegação principal"
      className={cn(
        'sticky bottom-0 z-30 flex items-stretch justify-around border-t border-border/60 bg-background/90 backdrop-blur-md md:hidden',
        className,
      )}
      style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
    >
      {primary.map(({ to, label, Icon }) => (
        <Link
          key={to}
          to={to}
          className={cn(
            'flex flex-1 flex-col items-center gap-0.5 px-2 py-2.5 text-[0.7rem] text-muted-foreground transition-colors',
            'hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60',
          )}
          activeProps={{
            className: 'text-[color:var(--primary)]',
          }}
        >
          <Icon className="size-5" aria-hidden />
          <span>{label}</span>
        </Link>
      ))}
    </nav>
  )
}

export { BottomNav }
