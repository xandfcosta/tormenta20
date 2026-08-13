import { render } from '@solidjs/testing-library'
import { RouterProvider, createMemoryHistory, createRootRoute, createRouter } from '@tanstack/solid-router'
import type { JSX } from 'solid-js'

/**
 * Mounts a component that renders `<Link>`s. TanStack's Link reads the router
 * from context and throws without one, so any scene fragment with navigation
 * needs this instead of a bare `render`.
 *
 * @example renderWithRouter(() => <MembersCard campaignId={7} isGm />)
 */
export function renderWithRouter(ui: () => JSX.Element) {
  const rootRoute = createRootRoute({ component: ui })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  return render(() => <RouterProvider router={router} />)
}
