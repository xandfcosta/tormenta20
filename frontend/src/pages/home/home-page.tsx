import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { PlayCircle, Scroll, Users2, Wand2 } from 'lucide-react'
import { meQueryOptions } from '@/entities/user/queries'
import { useLogout } from '@/entities/user/use-logout'
import { useActiveSession } from '@/features/session-resume/use-active-session'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useSceneNav } from '@/shared/lib/use-scene-nav'
import { useSfx } from '@/shared/lib/use-sfx'
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
  const activeSession = useActiveSession()
  const logout = useLogout(() => navigate({ to: '/login' }))
  const sfx = useSfx()
  const sfxEnabled = useUiStore((s) => s.sfx)
  const toggleSfx = useUiStore((s) => s.toggleSfx)
  const name = me.data?.name ?? me.data?.email ?? 'Aventureiro'

  // Play the select cue, then go — wraps each menu destination.
  const go = (select: () => void) => () => {
    sfx('select')
    select()
  }

  // Keyboard: the Hub is a single vertical menu region — ↑/↓ walk it, Enter
  // picks. It's the root scene, so Esc has nowhere to go back to (no-op).
  useSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]'),
    onEscape: () => {},
    sfx,
  })

  const items: HubMenuItem[] = [
    {
      label: 'Meus Heróis',
      icon: Users2,
      onSelect: go(() => navigate({ to: '/characters' })),
    },
    {
      label: 'Crônicas',
      icon: Scroll,
      onSelect: go(() => navigate({ to: '/campaigns' })),
    },
    {
      label: 'Ferramentas do Mestre',
      icon: Wand2,
      onSelect: go(() => navigate({ to: '/gm' })),
    },
  ]
  // Only surfaces while a session is live — a game's "Continue". Both GM and
  // players resume into the same match-mode screen (ALE-40).
  if (activeSession) {
    items.push({
      label: 'Continuar sessão',
      icon: PlayCircle,
      hasNext: true,
      onSelect: go(() =>
        navigate({
          to: '/campaigns/$id/sessions/$sid',
          params: {
            id: activeSession.campaignId,
            sid: activeSession.sessionId,
          },
        }),
      ),
    })
  }

  return (
    <SceneShell
      title="Tormenta 20"
      kicker="— Grimório de Arton —"
      onEnter={() => sfx('transition')}
    >
      <HubMenu
        className="mt-10"
        items={items}
        onItemHover={() => sfx('hover')}
      />
      <HubFooter
        className="mt-auto"
        name={name}
        onLogout={() => logout.mutate()}
        logoutPending={logout.isPending}
        sfxEnabled={sfxEnabled}
        onToggleSfx={toggleSfx}
      />
    </SceneShell>
  )
}
