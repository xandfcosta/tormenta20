/* @refresh reload */
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { RouterProvider, createRouter } from '@tanstack/solid-router'
import { render } from 'solid-js/web'
import { AuthProvider } from '@/shared/stores/auth-context'
import { UiProvider } from '@/shared/stores/ui-context'
import '@/index.css'
import { routeTree } from '../routeTree.gen'

const queryClient = new QueryClient()
const router = createRouter({ routeTree, context: { queryClient } })

declare module '@tanstack/solid-router' {
  interface Register {
    router: typeof router
  }
}

const root = document.getElementById('root')
if (!root) throw new Error('main: #root ausente no index.html (esperado <div id="root">)')

render(
  () => (
    <QueryClientProvider client={queryClient}>
      <UiProvider>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </UiProvider>
    </QueryClientProvider>
  ),
  root,
)
