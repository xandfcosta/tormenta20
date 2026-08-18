import { Link } from '@tanstack/solid-router'
import { LogOut, Volume2, VolumeX } from 'lucide-solid'
import { type JSX, Show, createSignal } from 'solid-js'
import { SceneContainerProvider } from '@/shared/lib/scene-container'
import { Button, buttonVariants } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'

/**
 * Full-screen frame for a live session ("match mode"): a slim bar with the
 * session identity, presence and the way out, over a scrollable body. The
 * role-specific views render as children.
 */
export function MatchShell(props: {
  campaignId: number
  title: string
  /** Right-hand slot — presence chips today. */
  bar?: JSX.Element
  /** O som se liga e desliga AQUI, e não só no Hub: mudar de ideia sobre áudio
   *  no meio de uma sessão ao vivo não pode custar sair da mesa (ALE-165). */
  sfxEnabled: boolean
  onToggleSfx: () => void
  children: JSX.Element
}) {
  // Publicado para os overlays (Dialog/Select/Popover) portarem para DENTRO da
  // cena. A partida carrega o escopo de tokens mas não o publicava, então todo
  // diálogo da sessão ao vivo — "Reiniciar o combate?", o ajuste de PV, o
  // descanso de dia — abria em shadcn CLARO sobre a mesa escura (ALE-122).
  const [sceneEl, setSceneEl] = createSignal<HTMLElement | null>(null)

  return (
    // `scene-grimorio` is the TOKEN SCOPE, not a skin: without it the whole
    // match renders in shadcn's light defaults. Match mode owns the viewport,
    // so it carries the scope itself instead of going through SceneShell,
    // which brings a back button and a title bar this screen already has.
    <div
      ref={setSceneEl}
      class="scene-grimorio flex h-dvh min-h-0 flex-col bg-background text-foreground"
    >
      <header class="flex items-center justify-between gap-3 border-b border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2 sm:px-4">
        <p class="min-w-0 flex-1 truncate font-heading tracking-wide text-grimorio-gold">
          {props.title}
        </p>
        <div class="flex items-center gap-2">
          {props.bar}
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={props.sfxEnabled ? 'Desligar o som' : 'Ligar o som'}
            aria-pressed={props.sfxEnabled}
            onClick={() => props.onToggleSfx()}
          >
            <Show when={props.sfxEnabled} fallback={<VolumeX aria-hidden="true" class="size-4" />}>
              <Volume2 aria-hidden="true" class="size-4" />
            </Show>
          </Button>
          {/* No `asChild` in Solid: a link that looks like a button IS a link
              wearing the button classes. */}
          <Link
            to="/campaigns/$id"
            params={{ id: String(props.campaignId) }}
            class={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'gap-1.5')}
          >
            <LogOut aria-hidden="true" class="size-4" />
            <span class="hidden sm:inline">Sair da sessão</span>
          </Link>
        </div>
      </header>
      <div class="min-h-0 flex-1 overflow-y-auto">
        <SceneContainerProvider element={sceneEl}>{props.children}</SceneContainerProvider>
      </div>
    </div>
  )
}
