import { useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { LogOut } from 'lucide-solid'
import { Show } from 'solid-js'
import { logout } from '@/entities/user/logout'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createSfx } from '@/shared/lib/sfx'
import { useAuth } from '@/shared/stores/auth-context'
import { useUi } from '@/shared/stores/ui-context'
import { GameMenuButton } from '@/shared/ui/game-menu-button'

/**
 * Placeholder for the real Hub (ALE-69). It exists to prove the foundation
 * end-to-end: the route guard let us in, the session survived the redirect,
 * the scene frame renders, and logging out sends us back to /login.
 */
export function FoundationHub() {
  const auth = useAuth()
  const ui = useUi()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const sfx = createSfx(ui)

  const signOut = async () => {
    await logout({ queryClient, auth })
    await navigate({ to: '/login' })
  }

  return (
    <SceneShell
      title="Tormenta 20"
      kicker="— Fundação Solid (ALE-64/66) —"
      onEnter={() => sfx('transition')}
    >
      <div class="mx-auto mt-10 flex w-full max-w-sm flex-col gap-4">
        <Show
          when={auth.user()}
          fallback={<p class="text-center text-muted-foreground">Carregando…</p>}
        >
          {(user) => (
            <p class="text-center text-sm text-muted-foreground">
              Sessão de <strong class="text-foreground">{user().name ?? user().email}</strong>
            </p>
          )}
        </Show>
        <GameMenuButton icon={LogOut} onClick={signOut}>
          Sair
        </GameMenuButton>
      </div>
    </SceneShell>
  )
}
