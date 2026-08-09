import { ChevronLeft } from 'lucide-react'
import type { ReactNode } from 'react'
import { usePrefersReducedMotion } from '@/shared/lib/use-media-query'
import { cn } from '@/shared/lib/utils'
import { BackgroundTexture } from '@/shared/ui/background-texture'
import { SceneTitle } from '@/shared/ui/scene-title'

/**
 * SceneShell — the frame every game scene lives in. It owns a fixed `dvh`
 * viewport (a scene doesn't scroll like a web document — its content column
 * scrolls inside instead), the `.scene-grimorio` token scope, the textured
 * backdrop, an optional Cinzel title, and an optional diegetic "back" control.
 *
 * The enter transition plays on mount — so navigating between scenes (each of
 * which mounts its own SceneShell) animates — and is skipped entirely under
 * `prefers-reduced-motion` (WCAG 2.3.3). Generalizes the app's `bare`
 * full-screen "match mode"; a scene page renders in a bare route and wraps its
 * body here. See [[reference_tanstack_nested_routes]] for the bare wiring.
 *
 * @example
 * <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —">
 *   <HubMenu />
 * </SceneShell>
 */
type SceneShellProps = {
  children: ReactNode
  /** Cinematic scene title (Cinzel). Omit for a title-less scene. */
  title?: ReactNode
  kicker?: ReactNode
  /** When set, renders a "back" control top-left that calls this. */
  onBack?: () => void
  backLabel?: string
  texture?: 'stone' | 'parchment'
  /** Extra classes for the scrollable content column. */
  className?: string
}

function SceneShell({
  children,
  title,
  kicker,
  onBack,
  backLabel = 'Voltar',
  texture = 'stone',
  className,
}: SceneShellProps) {
  const reduced = usePrefersReducedMotion()
  return (
    <section
      data-slot="scene-shell"
      className="scene-grimorio relative flex h-dvh flex-col overflow-hidden"
    >
      <BackgroundTexture variant={texture} vignette={texture === 'stone'} />
      {onBack ? (
        <button
          type="button"
          onClick={onBack}
          className="absolute left-3 top-3 z-10 inline-flex min-h-11 items-center gap-1 rounded-sm px-3 font-heading text-sm tracking-wide text-muted-foreground transition-colors hover:text-grimorio-gold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold"
          style={{ paddingTop: 'max(0.75rem, env(safe-area-inset-top))' }}
        >
          <ChevronLeft aria-hidden className="size-4" />
          {backLabel}
        </button>
      ) : null}
      <div
        data-slot="scene-content"
        data-animate={reduced ? undefined : true}
        className={cn(
          'relative flex min-h-0 flex-1 flex-col overflow-y-auto px-5 py-14',
          !reduced && 'scene-in',
          className,
        )}
      >
        {title ? <SceneTitle kicker={kicker}>{title}</SceneTitle> : null}
        {children}
      </div>
    </section>
  )
}

export { SceneShell }
