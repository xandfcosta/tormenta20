import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import '../index.css'
import { RoutePendingSkeleton } from '../shared/ui/skeleton'
import { routeTree } from '../routeTree.gen'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000 } },
})

const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
  // Blocking loaders keep the old page frozen while awaiting the first
  // fetch; show a skeleton after 150ms instead of that dead gap.
  defaultPendingComponent: RoutePendingSkeleton,
  defaultPendingMs: 150,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)
