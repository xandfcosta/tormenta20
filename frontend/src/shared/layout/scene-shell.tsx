import { ChevronLeft } from 'lucide-react'
import { type ReactNode, useEffect, useRef } from 'react'
import { usePrefersReducedMotion } from '@/shared/lib/use-media-query'
import { cn } from '@/shared/lib/utils'
import { BackgroundTexture } from '@/shared/ui/background-texture'
import { SceneTitle } from '@/shared/ui/scene-title'

/**
 * SceneShell — the frame every game scene lives in. It owns a fixed `dvh`
 * viewport (a scene doesn't scroll like a web document — its content column
 * scrolls inside instead), the `.scene-grimorio` token scope, the textured
 * backdrop, an optional title, and an optional diegetic "back" control.
 *
 * Two layouts: the default (Hub-style) centers a big cinematic Cinzel title in
 * the content; `dense` (content-heavy section scenes like the roster) uses a
 * compact top header row — back + small title + `headerRight` — and tighter
 * padding so the stage/list keeps the height.
 *
 * The enter transition plays on mount and is skipped under
 * `prefers-reduced-motion` (WCAG 2.3.3). Generalizes the app's `bare`
 * full-screen "match mode"; a scene page renders in a bare route and wraps its
 * body here. See [[reference_tanstack_nested_routes]] for the bare wiring.
 *
 * @example
 * <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —"><HubMenu /></SceneShell>
 * <SceneShell dense title="Personagens" onBack={toHub} headerRight={<Search/>}>…</SceneShell>
 */
type SceneShellProps = {
  children: ReactNode
  title?: ReactNode
  kicker?: ReactNode
  /** When set, renders a diegetic "back" control that calls this. */
  onBack?: () => void
  backLabel?: string
  texture?: 'stone' | 'parchment'
  /** Fires once when the scene mounts *and* motion is allowed — pairs the
   *  enter animation with an optional cue (e.g. a transition sound). */
  onEnter?: () => void
  /** Compact header-row layout for content-heavy section scenes. */
  dense?: boolean
  /** Controls aligned to the right of the dense header (search, actions). */
  headerRight?: ReactNode
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
  onEnter,
  dense = false,
  headerRight,
  className,
}: SceneShellProps) {
  const reduced = usePrefersReducedMotion()
  // Fire onEnter once on mount (a scene "enters" once), gated by motion. Refs
  // keep the mount-only effect free of reactive deps.
  const enterRef = useRef(onEnter)
  enterRef.current = onEnter
  const reducedRef = useRef(reduced)
  reducedRef.current = reduced
  useEffect(() => {
    if (!reducedRef.current) enterRef.current?.()
  }, [])

  return (
    <section
      data-slot="scene-shell"
      data-dense={dense || undefined}
      className="scene-grimorio relative flex h-dvh flex-col overflow-hidden"
    >
      <BackgroundTexture variant={texture} vignette={texture === 'stone'} />

      {dense ? (
        <header
          className="relative flex flex-wrap items-center gap-3 border-b border-grimorio-iron px-4 py-3"
          style={{ paddingTop: 'max(0.75rem, env(safe-area-inset-top))' }}
        >
          {onBack ? <BackControl onBack={onBack} label={backLabel} /> : null}
          {title ? (
            <h1 className="font-heading text-xl tracking-wide text-foreground">
              {title}
            </h1>
          ) : null}
          {headerRight ? (
            <div className="ml-auto flex flex-1 flex-wrap items-center justify-end gap-2">
              {headerRight}
            </div>
          ) : null}
        </header>
      ) : onBack ? (
        <div
          className="absolute left-3 top-3 z-10"
          style={{ paddingTop: 'env(safe-area-inset-top)' }}
        >
          <BackControl onBack={onBack} label={backLabel} />
        </div>
      ) : null}

      <div
        data-slot="scene-content"
        data-animate={reduced ? undefined : true}
        className={cn(
          // overflow-x-hidden: a scene never scrolls horizontally, and it
          // clips slide-in overlays (the dossier) so they don't flash a
          // transient bottom scrollbar mid-animation.
          'relative flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto',
          dense ? 'px-4 py-4' : 'px-5 py-14',
          !reduced && 'scene-in',
          className,
        )}
      >
        {!dense && title ? (
          <SceneTitle kicker={kicker}>{title}</SceneTitle>
        ) : null}
        {children}
      </div>
    </section>
  )
}

/** The diegetic back affordance — shared by both layouts. */
function BackControl({ onBack, label }: { onBack: () => void; label: string }) {
  return (
    <button
      type="button"
      onClick={onBack}
      className="inline-flex min-h-11 items-center gap-1 rounded-sm px-3 font-heading text-sm tracking-wide text-muted-foreground transition-colors hover:text-grimorio-gold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold"
    >
      <ChevronLeft aria-hidden className="size-4" />
      {label}
    </button>
  )
}

export { SceneShell }
