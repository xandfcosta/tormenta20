import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRouter } from '@tanstack/react-router'
/* Self-hosted fonts. Cinzel = inscriptional display face for headings
 * (replaced Redaction — that degraded face read poorly on screen at UI
 * sizes); EB Garamond = readable historical serif for body; JetBrains
 * Mono = numeric/HUD. Variable axes keep the bundle small. */
import '@fontsource-variable/cinzel/index.css'
import '@fontsource-variable/eb-garamond/index.css'
import '@fontsource-variable/jetbrains-mono/index.css'
import '../index.css'
import { routeTree } from '../routeTree.gen'

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 30_000 } },
})

const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
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
