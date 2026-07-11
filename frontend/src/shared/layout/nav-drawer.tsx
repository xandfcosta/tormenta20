import { Link } from '@tanstack/react-router'
import { LogOut, Moon, Sun } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/shared/ui/sheet'
import { cn } from '@/shared/lib/utils'
import type { NavDestination } from './nav-destinations'

/**
 * Mobile drawer wrapping the full nav — includes primary + secondary
 * destinations (bottom nav only shows primary), theme toggle, and
 * user session actions. Controlled from `AppShell` so opening it
 * doesn't require touching global state.
 */
type NavDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  destinations: readonly NavDestination[]
  user: { name?: string | null; email: string } | null
  theme: 'light' | 'dark'
  onToggleTheme: () => void
  onLogout: () => void
  logoutPending?: boolean
}

function NavDrawer({
  open,
  onOpenChange,
  destinations,
  user,
  theme,
  onToggleTheme,
  onLogout,
  logoutPending,
}: NavDrawerProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="left" size="md" className="gap-6">
        <SheetHeader>
          <SheetTitle>Tormenta 20</SheetTitle>
          <SheetDescription>
            {user ? (user.name ?? user.email) : 'Convidado'}
          </SheetDescription>
        </SheetHeader>

        <nav className="flex flex-col gap-1" aria-label="Menu">
          {destinations.map(({ to, label, Icon }) => (
            <Link
              key={to}
              to={to}
              onClick={() => onOpenChange(false)}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm text-foreground/80 transition-colors hover:bg-secondary hover:text-foreground',
              )}
              activeProps={{
                className:
                  'bg-secondary text-foreground',
              }}
            >
              <Icon className="size-4 shrink-0" aria-hidden />
              <span>{label}</span>
            </Link>
          ))}
        </nav>

        <div className="mt-auto flex flex-col gap-2">
          <Button
            variant="outline"
            size="sm"
            className="justify-start gap-2"
            onClick={onToggleTheme}
          >
            {theme === 'light' ? (
              <Moon className="size-4" />
            ) : (
              <Sun className="size-4" />
            )}
            {theme === 'light' ? 'Modo escuro' : 'Modo claro'}
          </Button>
          {user ? (
            <Button
              variant="ghost"
              size="sm"
              className="justify-start gap-2"
              onClick={onLogout}
              disabled={logoutPending}
            >
              <LogOut className="size-4" />
              Sair
            </Button>
          ) : (
            <div className="flex gap-2">
              <Link
                to="/login"
                className="flex-1"
                onClick={() => onOpenChange(false)}
              >
                <Button variant="outline" size="sm" className="w-full">
                  Entrar
                </Button>
              </Link>
              <Link
                to="/register"
                className="flex-1"
                onClick={() => onOpenChange(false)}
              >
                <Button size="sm" className="w-full">
                  Registrar
                </Button>
              </Link>
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

export { NavDrawer }
