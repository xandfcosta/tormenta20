import { ChevronLeft } from 'lucide-solid'
import { type JSX, type ParentProps, Show, createSignal, onMount } from 'solid-js'
import { SceneContainerProvider } from '@/shared/lib/scene-container'
import { createPrefersReducedMotion } from '@/shared/lib/media-query'
import { cn } from '@/shared/lib/utils'
import { BackgroundTexture } from '@/shared/ui/background-texture'
import { SceneTitle } from '@/shared/ui/scene-title'

export type SceneShellProps = ParentProps<{
  title?: JSX.Element
  kicker?: JSX.Element
  /** When set, renders a diegetic "back" control that calls this. */
  onBack?: () => void
  backLabel?: string
  texture?: 'stone' | 'parchment'
  /** Override the decorative edge vignette (defaults on for stone). Scenes that
   *  drive their own edge effect — the sheet's HP vignette — pass `false`. */
  vignette?: boolean
  /** Fires once when the scene mounts *and* motion is allowed — pairs the enter
   *  animation with an optional cue (e.g. a transition sound). */
  onEnter?: () => void
  /** Compact header-row layout for content-heavy section scenes. */
  dense?: boolean
  /** Controls aligned to the right of the dense header (search, actions). */
  headerRight?: JSX.Element
  /** Full-bleed content: no padding, no outer scroll — the child owns its own
   *  height/scroll (e.g. the character sheet's HUD-pinned grid). */
  bleed?: boolean
  /** Extra classes for the scrollable content column. */
  class?: string
}>

/**
 * The frame every game scene lives in. It owns a fixed `dvh` viewport (a scene
 * doesn't scroll like a web document — its content column scrolls inside
 * instead), the `.scene-grimorio` token scope, the textured backdrop, an
 * optional title, and an optional diegetic "back" control.
 *
 * Two layouts: the default (Hub-style) centers a big cinematic Cinzel title;
 * `dense` (content-heavy section scenes like the roster) uses a compact top
 * header row — back + small title + `headerRight` — and tighter padding so the
 * stage keeps the height.
 *
 * The enter transition plays on mount and is skipped under
 * `prefers-reduced-motion` (WCAG 2.3.3).
 *
 * @example
 * <SceneShell title="Tormenta 20" kicker="— Grimório de Arton —"><HubMenu /></SceneShell>
 * <SceneShell dense title="Personagens" onBack={toHub} headerRight={<Search/>}>…</SceneShell>
 */
export function SceneShell(props: SceneShellProps) {
  const reduced = createPrefersReducedMotion()
  // Published so overlays (Dialog/Popover/Tooltip/Select) can portal into the
  // scene and inherit `.scene-grimorio` instead of rendering shadcn over body.
  const [sceneEl, setSceneEl] = createSignal<HTMLElement | null>(null)

  // A scene "enters" once. onMount is inherently mount-only, so the React
  // version's ref dance to keep the effect dep-free simply isn't needed.
  onMount(() => {
    if (!reduced()) props.onEnter?.()
  })

  const texture = () => props.texture ?? 'stone'

  return (
    <SceneContainerProvider element={sceneEl}>
      <section
        ref={setSceneEl}
        data-slot="scene-shell"
        data-dense={props.dense || undefined}
        class="scene-grimorio relative flex h-dvh flex-col overflow-hidden"
      >
        <BackgroundTexture variant={texture()} vignette={props.vignette ?? texture() === 'stone'} />

        <Show
          when={props.dense}
          fallback={
            <Show when={props.onBack}>
              {(onBack) => (
                <div class="absolute left-3 top-3 z-10" style={{ 'padding-top': 'env(safe-area-inset-top)' }}>
                  <BackControl onBack={onBack()} label={props.backLabel ?? 'Voltar'} />
                </div>
              )}
            </Show>
          }
        >
          <header
            class="relative flex flex-wrap items-center gap-3 border-b border-grimorio-iron px-4 py-3"
            style={{ 'padding-top': 'max(0.75rem, env(safe-area-inset-top))' }}
          >
            <Show when={props.onBack}>
              {(onBack) => <BackControl onBack={onBack()} label={props.backLabel ?? 'Voltar'} />}
            </Show>
            <Show when={props.title}>
              {(title) => <h1 class="font-heading text-xl tracking-wide text-foreground">{title()}</h1>}
            </Show>
            <Show when={props.headerRight}>
              {(right) => (
                <div class="ml-auto flex flex-1 flex-wrap items-center justify-end gap-2">{right()}</div>
              )}
            </Show>
          </header>
        </Show>

        <div
          data-slot="scene-content"
          data-animate={reduced() ? undefined : true}
          class={cn(
            'relative flex min-h-0 flex-1 flex-col',
            // A scene never scrolls horizontally (overflow-x-hidden also clips
            // slide-in overlays like the dossier). `bleed` hands scroll +
            // padding to a full-height child; otherwise the column scrolls.
            props.bleed
              ? 'overflow-hidden'
              : cn('overflow-x-hidden overflow-y-auto', props.dense ? 'px-4 py-4' : 'px-5 py-14'),
            !reduced() && 'scene-in',
            props.class,
          )}
        >
          <Show when={!props.dense && props.title}>
            {(title) => <SceneTitle kicker={props.kicker}>{title()}</SceneTitle>}
          </Show>
          {props.children}
        </div>
      </section>
    </SceneContainerProvider>
  )
}

/** The diegetic back affordance — shared by both layouts. */
function BackControl(props: { onBack: () => void; label: string }) {
  return (
    <button
      type="button"
      onClick={() => props.onBack()}
      class="inline-flex min-h-11 items-center gap-1 rounded-none px-3 font-heading text-sm tracking-wide text-muted-foreground transition-colors hover:text-grimorio-gold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-grimorio-gold"
    >
      <ChevronLeft aria-hidden="true" class="size-4" />
      {props.label}
    </button>
  )
}
