/* @refresh reload */
import { QueryClient, QueryClientProvider } from '@tanstack/solid-query'
import { RouterProvider, createRouter } from '@tanstack/solid-router'
import { render } from 'solid-js/web'
import { AuthProvider } from '@/shared/stores/auth-context'
import { ConditionalsProvider } from '@/shared/stores/conditionals-context'
import { PowerUsesProvider } from '@/shared/stores/power-uses-context'
import { StanceActivationProvider } from '@/shared/stores/stance-activation-context'
import { UiProvider } from '@/shared/stores/ui-context'
import '@/index.css'
import { routeTree } from '../routeTree.gen'

const queryClient = new QueryClient()
// `intent`: passar o cursor (ou o foco pelo teclado) em um link já busca o
// chunk da cena e roda o loader dela, então o clique não paga mais esse tempo.
// O atraso evita disparar em link que o mouse só atravessou.
// `intent`: passar o cursor (ou o foco pelo teclado) em um link já busca o
// chunk da cena e roda o loader dela, então o clique não paga mais esse tempo.
// O atraso evita disparar em link que o mouse só atravessou.
const router = createRouter({
  routeTree,
  context: { queryClient },
  defaultPreload: 'intent',
  defaultPreloadDelay: 60,
})

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
          <ConditionalsProvider>
            <PowerUsesProvider>
              <StanceActivationProvider>
                <RouterProvider router={router} />
              </StanceActivationProvider>
            </PowerUsesProvider>
          </ConditionalsProvider>
        </AuthProvider>
      </UiProvider>
    </QueryClientProvider>
  ),
  root,
)
