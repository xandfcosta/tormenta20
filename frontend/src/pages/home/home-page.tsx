import { useQueryClient, useQuery } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { PlayCircle, Scroll, Users2, Wand2 } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { logout } from '@/entities/user/logout'
import { meQueryOptions } from '@/entities/user/queries'
import { InvitePlayer } from '@/features/account-invite/invite-player-dialog'
import { createActiveSession } from '@/features/session-resume/active-session'
import { createLiveSessionPrefetch } from '@/features/session-resume/prefetch-live-session'
import { SceneShell } from '@/shared/layout/scene-shell'
import { createFullscreen } from '@/shared/lib/fullscreen'
import { createSceneNav } from '@/shared/lib/scene-nav'
import { createSfx, createSfxToggle } from '@/shared/lib/sfx'
import { useAuth } from '@/shared/stores/auth-context'
import { useUi } from '@/shared/stores/ui-context'
import { HubFooter } from './hub-footer'
import { type HubMenuItem, HubMenu } from './hub-menu'

/**
 * The Hub: the app's main menu, rendered as a full game scene. Three entries
 * into the main areas, "Continuar sessão" while a session is live, and the
 * player's identity footer.
 */
export function HomePage() {
  const navigate = useNavigate()
  const auth = useAuth()
  const ui = useUi()
  const queryClient = useQueryClient()
  const me = useQuery(() => meQueryOptions)
  const activeSession = createActiveSession()
  // A cena da partida é carregada enquanto o jogador ainda olha o menu, porque
  // "Continuar sessão" é a ação primária e o trabalho dela é conhecido antes
  // do clique.
  createLiveSessionPrefetch(activeSession)
  // A cena da partida é carregada enquanto o jogador ainda olha o menu, porque
  // "Continuar sessão" é a ação primária e o trabalho dela é conhecido antes
  // do clique.
  const sfx = createSfx(ui)
  const toggleSfx = createSfxToggle(ui, sfx)
  const fullscreen = createFullscreen()
  const [signingOut, setSigningOut] = createSignal(false)
  const [inviting, setInviting] = createSignal(false)

  const name = () => me.data?.name ?? me.data?.email ?? 'Aventureiro'

  // Play the select cue, then go — wraps each menu destination.
  const go = (select: () => void) => () => {
    sfx('select')
    select()
  }

  // Keyboard: the Hub is a single vertical menu region — ↑/↓ walk it, Enter
  // picks. It's the root scene, so Esc has nowhere to go back to (no-op).
  createSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]'),
    onEscape: () => {},
    sfx,
  })

  const signOut = async () => {
    setSigningOut(true)
    try {
      await logout({ queryClient, auth })
      await navigate({ to: '/login' })
    } finally {
      setSigningOut(false)
    }
  }

  const items = (): HubMenuItem[] => {
    const base: HubMenuItem[] = [
      { label: 'Meus Heróis', icon: Users2, onSelect: go(() => navigate({ to: '/characters' })) },
      { label: 'Crônicas', icon: Scroll, onSelect: go(() => navigate({ to: '/campaigns' })) },
      { label: 'Ferramentas do Mestre', icon: Wand2, onSelect: go(() => navigate({ to: '/gm' })) },
    ]
    // Only surfaces while a session is live — a game's "Continue". Both GM and
    // players resume into the same match-mode screen.
    const live = activeSession()
    if (live) {
      base.push({
        label: 'Continuar sessão',
        icon: PlayCircle,
        hasNext: true,
        onSelect: go(() =>
          navigate({
            to: '/campaigns/$id/sessions/$sid',
            params: { id: String(live.campaignId), sid: String(live.sessionId) },
          }),
        ),
      })
    }
    return base
  }

  return (
    <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —" onEnter={() => sfx('transition')}>
      <HubMenu class="mt-10" items={items()} onItemHover={() => sfx('hover')} />
      <HubFooter
        class="mt-auto"
        name={name()}
        onLogout={signOut}
        logoutPending={signingOut()}
        sfxEnabled={ui.sfx()}
        onToggleSfx={toggleSfx}
        fullscreenSupported={fullscreen.supported}
        fullscreenActive={fullscreen.active()}
        onToggleFullscreen={fullscreen.toggle}
        canInvite={me.data?.isAdmin}
        onInvite={() => setInviting(true)}
        canAdminister={me.data?.isAdmin}
        onAdminister={() => navigate({ to: '/admin' })}
      />
      <InvitePlayer open={inviting()} onOpenChange={setInviting} />
    </SceneShell>
  )
}
