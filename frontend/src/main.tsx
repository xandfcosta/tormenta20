import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createRouter } from '@tanstack/react-router'
/* Self-hosted fonts (Controlled Decay design system). Redaction =
 * display face with intentional degradation; EB Garamond = readable
 * historical serif for body; JetBrains Mono = numeric/HUD. Variable
 * axes on Garamond + Mono keep the bundle small. */
import '@fontsource/redaction/400.css'
import '@fontsource/redaction/700.css'
import '@fontsource-variable/eb-garamond/index.css'
import '@fontsource-variable/jetbrains-mono/index.css'
import './index.css'
import { routeTree } from './routeTree.gen'

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
