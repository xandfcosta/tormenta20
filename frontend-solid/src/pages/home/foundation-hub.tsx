import { useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { Show } from 'solid-js'
import { logout } from '@/entities/user/logout'
import { useAuth } from '@/shared/stores/auth-context'

/**
 * Placeholder for the real Hub (ALE-69). It exists to prove the foundation
 * end-to-end: the route guard let us in, the session survived the redirect,
 * and logging out sends us back to /login.
 */
export function FoundationHub() {
  const auth = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const signOut = async () => {
    await logout({ queryClient, auth })
    await navigate({ to: '/login' })
  }

  return (
    <main class="mx-auto flex min-h-dvh max-w-md flex-col justify-center gap-6 px-6">
      <div class="space-y-1.5">
        <p class="text-sm text-muted-foreground">Fundação Solid (ALE-64)</p>
        <h1 class="text-2xl font-semibold tracking-tight">Sessão autenticada</h1>
      </div>
      <Show when={auth.user()} fallback={<p class="text-muted-foreground">Carregando…</p>}>
        {(user) => (
          <p class="text-sm">
            Logado como <strong>{user().name ?? user().email}</strong> ({user().email})
          </p>
        )}
      </Show>
      <button
        type="button"
        onClick={signOut}
        class="h-9 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent"
      >
        Sair
      </button>
    </main>
  )
}
