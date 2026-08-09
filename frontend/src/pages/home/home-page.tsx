import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Scroll, Users2, Wand2 } from 'lucide-react'
import { meQueryOptions } from '@/entities/user/queries'
import { useLogout } from '@/entities/user/use-logout'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useUiStore } from '@/shared/stores/ui-store'
import { HubFooter } from './hub-footer'
import { HubMenu, type HubMenuItem } from './hub-menu'

/**
 * HomePage — the Hub: the app's main menu, rendered as a full game scene (a
 * bare route with no site nav — see the `inHub` bare-wiring in __root). Three
 * entries into the main areas plus the player's identity footer. "Continuar
 * sessão" (ALE-40) and the portrait quick-menu (ALE-39) slot in here next.
 */
export function HomePage() {
  const navigate = useNavigate()
  const me = useQuery(meQueryOptions)
  const theme = useUiStore((s) => s.theme)
  const toggleTheme = useUiStore((s) => s.toggleTheme)
  const logout = useLogout(() => navigate({ to: '/login' }))
  const name = me.data?.name ?? me.data?.email ?? 'Aventureiro'

  const items: HubMenuItem[] = [
    {
      label: 'Meus Heróis',
      icon: Users2,
      onSelect: () => navigate({ to: '/characters' }),
    },
    {
      label: 'Crônicas',
      icon: Scroll,
      onSelect: () => navigate({ to: '/campaigns' }),
    },
    {
      label: 'Ferramentas do Mestre',
      icon: Wand2,
      onSelect: () => navigate({ to: '/gm' }),
    },
  ]

  return (
    <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —">
      <HubMenu items={items} />
      <HubFooter
        className="mt-auto"
        name={name}
        theme={theme}
        onToggleTheme={toggleTheme}
        onLogout={() => logout.mutate()}
        logoutPending={logout.isPending}
      />
    </SceneShell>
  )
}
