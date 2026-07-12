import { useState, type ReactNode } from 'react'
import { BottomNav } from './bottom-nav'
import { NavDrawer } from './nav-drawer'
import { TopNav } from './top-nav'
import {
  AUTHED_DESTINATIONS,
  PUBLIC_DESTINATIONS,
  type NavDestination,
} from './nav-destinations'

/**
 * AppShell — the responsive frame shared by every route.
 *
 * Layout is `flex-col` on a `dvh` container so the main area
 * consumes the space between top + bottom nav even as mobile browser
 * chrome collapses. Bottom nav mounts on phone-tier viewports only;
 * desktop routes rely on the horizontal top nav.
 *
 * The shell doesn't own routing data — it takes the user / theme /
 * logout callback from the root layout so this stays a pure UI
 * component and can be tested without a router.
 */
type AppShellProps = {
  user: { name?: string | null; email: string } | null
  theme: 'light' | 'dark'
  onToggleTheme: () => void
  onLogout: () => void
  logoutPending?: boolean
  /** Full-screen "match" mode — drops the top/bottom nav + drawer so a
   * live session owns the whole viewport with its own session bar. */
  bare?: boolean
  children: ReactNode
}

function AppShell({
  user,
  theme,
  onToggleTheme,
  onLogout,
  logoutPending,
  bare,
  children,
}: AppShellProps) {
  const [drawerOpen, setDrawerOpen] = useState(false)
  const destinations: readonly NavDestination[] = user
    ? AUTHED_DESTINATIONS
    : PUBLIC_DESTINATIONS

  if (bare) {
    return (
      <div
        data-slot="app-shell-bare"
        className="flex h-dvh flex-col overflow-hidden bg-background text-foreground"
      >
        {children}
      </div>
    )
  }

  return (
    <div
      data-slot="app-shell"
      className="flex min-h-dvh flex-col overflow-x-hidden bg-background text-foreground"
    >
      <TopNav
        destinations={destinations}
        user={user}
        theme={theme}
        onToggleTheme={onToggleTheme}
        onLogout={onLogout}
        logoutPending={logoutPending}
        onOpenDrawer={() => setDrawerOpen(true)}
      />

      <main className="flex min-h-0 flex-1 flex-col overflow-x-hidden">
        {children}
      </main>

      {user && <BottomNav destinations={destinations} />}

      <NavDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        destinations={destinations}
        user={user}
        theme={theme}
        onToggleTheme={onToggleTheme}
        onLogout={onLogout}
        logoutPending={logoutPending}
      />
    </div>
  )
}

export { AppShell }
