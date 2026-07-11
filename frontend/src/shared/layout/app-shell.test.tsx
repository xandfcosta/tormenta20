import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from '@tanstack/react-router'
import { render, screen, type RenderResult } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { AppShell } from './app-shell'
import { BottomNav } from './bottom-nav'
import { AUTHED_DESTINATIONS, PUBLIC_DESTINATIONS } from './nav-destinations'

/**
 * Route wrapper — TanStack Router requires a Router context for
 * `<Link>` to render. We spin up a memory-history router with a
 * catch-all root route so the tests can exercise nav behavior
 * without matching real app routes.
 */
function renderWithRouter(node: ReactNode): RenderResult {
  const rootRoute = createRootRoute({ component: () => <>{node}</> })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  const qc = new QueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

const noop = () => {}

describe('AppShell', () => {
  it('renders children inside the shell', async () => {
    renderWithRouter(
      <AppShell
        user={null}
        theme="dark"
        onToggleTheme={noop}
        onLogout={noop}
      >
        <p>page body</p>
      </AppShell>,
    )
    expect(await screen.findByText('page body')).toBeInTheDocument()
    expect(document.querySelector('[data-slot="app-shell"]')).toBeTruthy()
  })

  it('hides the bottom nav when the user is unauthenticated', async () => {
    renderWithRouter(
      <AppShell
        user={null}
        theme="dark"
        onToggleTheme={noop}
        onLogout={noop}
      >
        <p>anon</p>
      </AppShell>,
    )
    await screen.findByText('anon')
    expect(document.querySelector('[data-slot="bottom-nav"]')).toBeNull()
  })

  it('shows the bottom nav for an authenticated user', async () => {
    renderWithRouter(
      <AppShell
        user={{ email: 'a@b.c' }}
        theme="dark"
        onToggleTheme={noop}
        onLogout={noop}
      >
        <p>me</p>
      </AppShell>,
    )
    await screen.findByText('me')
    expect(document.querySelector('[data-slot="bottom-nav"]')).toBeTruthy()
  })

  it('opens the mobile drawer when the hamburger fires', async () => {
    const user = userEvent.setup()
    renderWithRouter(
      <AppShell
        user={{ email: 'a@b.c' }}
        theme="dark"
        onToggleTheme={noop}
        onLogout={noop}
      >
        <p>me</p>
      </AppShell>,
    )
    const hamburger = await screen.findByRole('button', { name: 'Abrir menu' })
    await user.click(hamburger)
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
  })

  it('invokes onToggleTheme from the top nav toggle', async () => {
    const toggle = vi.fn()
    const user = userEvent.setup()
    renderWithRouter(
      <AppShell
        user={null}
        theme="dark"
        onToggleTheme={toggle}
        onLogout={noop}
      >
        <p>x</p>
      </AppShell>,
    )
    const themeBtn = await screen.findByRole('button', { name: 'Modo claro' })
    await user.click(themeBtn)
    expect(toggle).toHaveBeenCalledOnce()
  })
})

describe('BottomNav', () => {
  it('renders every primary destination in the authed list', async () => {
    renderWithRouter(<BottomNav destinations={AUTHED_DESTINATIONS} />)
    const primary = AUTHED_DESTINATIONS.filter((d) => d.tier === 'primary')
    expect(primary.length).toBeLessThanOrEqual(4)
    for (const d of primary) {
      expect(await screen.findByText(d.label)).toBeInTheDocument()
    }
  })

  it('skips secondary destinations', async () => {
    renderWithRouter(<BottomNav destinations={AUTHED_DESTINATIONS} />)
    /* Wait for at least one primary to hydrate so we know the mount
     * finished before asserting negatives. */
    const firstPrimary = AUTHED_DESTINATIONS.find((d) => d.tier === 'primary')!
    await screen.findByText(firstPrimary.label)
    const secondary = AUTHED_DESTINATIONS.filter((d) => d.tier === 'secondary')
    for (const d of secondary) {
      expect(screen.queryByText(d.label)).not.toBeInTheDocument()
    }
  })

  it('renders nothing when there are no primary destinations', () => {
    renderWithRouter(<BottomNav destinations={[]} />)
    expect(document.querySelector('[data-slot="bottom-nav"]')).toBeNull()
  })

  it('keeps PUBLIC_DESTINATIONS in the primary tier only', () => {
    // Sanity guard so a future edit doesn't accidentally push a
    // secondary tier item into PUBLIC_DESTINATIONS (which BottomNav
    // would silently skip).
    for (const d of PUBLIC_DESTINATIONS) {
      expect(d.tier).toBe('primary')
    }
  })
})
