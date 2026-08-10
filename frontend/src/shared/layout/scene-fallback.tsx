import { BackgroundTexture } from '@/shared/ui/background-texture'

/**
 * SceneFallback — the router `pendingComponent` for bare grimório scene routes.
 *
 * A scene route renders on the app's bare shell (no AppShell chrome), so while
 * its blocking loader or code-split chunk resolves the generic white
 * `RoutePendingSkeleton` would flash the light `--background` between two dark
 * scenes — the "load em branco" going roster → ficha. This paints the SAME
 * `.scene-grimorio` dark textured ground a real SceneShell does, so the
 * transition stays dark end-to-end; a faint centered pulse signals the wait.
 */
function SceneFallback() {
  return (
    <div className="scene-grimorio relative flex h-dvh items-center justify-center overflow-hidden">
      <BackgroundTexture variant="stone" vignette />
      <div
        aria-hidden
        className="size-2 animate-pulse rounded-full bg-grimorio-gold/70"
      />
      <span className="sr-only">Carregando…</span>
    </div>
  )
}

export { SceneFallback }
