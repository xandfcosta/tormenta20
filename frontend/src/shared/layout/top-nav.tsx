import { Link } from '@tanstack/react-router'
import { Moon, Sun } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import type { NavDestination } from './nav-destinations'

/**
 * Desktop header nav — brand mark on the left, destinations in the
 * middle (only when authed + on md+), user chip + theme toggle on the
 * right. Mobile still gets a compact version of this via the same
 * component: hamburger trigger + brand only, everything else hidden.
 */
type TopNavProps = {
  brand?: string
  destinations: readonly NavDestination[]
  user: { name?: string | null; email: string } | null
  theme: 'light' | 'dark'
  onToggleTheme: () => void
  onLogout: () => void
  logoutPending?: boolean
  onOpenDrawer: () => void
}

function TopNav({
  brand = 'Tormenta 20',
  destinations,
  user,
  theme,
  onToggleTheme,
  onLogout,
  logoutPending,
  onOpenDrawer,
}: TopNavProps) {
  return (
    <header
      data-slot="top-nav"
      className="sticky top-0 z-30 flex items-center justify-between gap-3 border-b border-border/60 bg-background/85 px-4 py-2.5 backdrop-blur-md md:px-6 md:py-3"
      style={{ paddingTop: 'max(0.625rem, env(safe-area-inset-top))' }}
    >
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="sm"
          className="md:hidden"
          onClick={onOpenDrawer}
          aria-label="Abrir menu"
        >
          <span aria-hidden className="text-lg leading-none">☰</span>
        </Button>
        <Link
          to="/"
          className="font-display text-lg tracking-widest text-foreground hover:text-[color:var(--primary)]"
        >
          {brand}
        </Link>
      </div>

      <nav className="hidden md:flex md:items-center md:gap-1">
        {destinations.map((d) => (
          <Link
            key={d.to}
            to={d.to}
            className={cn(
              'rounded-md px-3 py-1.5 text-sm text-foreground/80 transition-colors hover:bg-secondary hover:text-foreground',
            )}
            activeProps={{ className: 'bg-secondary text-foreground' }}
          >
            {d.label}
          </Link>
        ))}
      </nav>

      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={onToggleTheme}
          aria-label={theme === 'light' ? 'Modo escuro' : 'Modo claro'}
        >
          {theme === 'light' ? (
            <Moon className="size-4" />
          ) : (
            <Sun className="size-4" />
          )}
        </Button>
        {user ? (
          <>
            <span className="hidden max-w-[14ch] truncate text-sm text-muted-foreground sm:inline">
              {user.name ?? user.email}
            </span>
            <Button
              size="sm"
              variant="outline"
              onClick={onLogout}
              disabled={logoutPending}
              className="hidden sm:inline-flex"
            >
              Sair
            </Button>
          </>
        ) : (
          <>
            <Link to="/login">
              <Button variant="outline" size="sm">
                Entrar
              </Button>
            </Link>
            <Link to="/register" className="hidden sm:inline-flex">
              <Button size="sm">Registrar</Button>
            </Link>
          </>
        )}
      </div>
    </header>
  )
}

export { TopNav }
