import { Dynamic } from 'solid-js/web'
import { type ComponentProps, type JSX, Show, splitProps } from 'solid-js'
import { cn } from '@/shared/lib/utils'

export type SceneTitleProps = ComponentProps<'h1'> & {
  as?: 'h1' | 'h2'
  kicker?: JSX.Element
}

/**
 * A cinematic title for the Grimório scenes: Cinzel caps, wide tracking and a
 * soft arcane glow. An optional `kicker` renders a small gold eyebrow beneath.
 * Meant to live inside a `.scene-grimorio` scope.
 *
 * @example <SceneTitle kicker="— Grimório de Arton —">Tormenta 20</SceneTitle>
 */
export function SceneTitle(props: SceneTitleProps) {
  const [local, rest] = splitProps(props, ['class', 'children', 'kicker', 'as'])
  return (
    <div data-slot="scene-title" class="flex flex-col items-center gap-1 text-center">
      <Dynamic
        component={local.as ?? 'h1'}
        class={cn(
          'scene-title-glow font-heading font-bold text-[clamp(2.2rem,8vw,3.4rem)] uppercase leading-tight tracking-[0.18em] text-foreground',
          local.class,
        )}
        {...rest}
      >
        {local.children}
      </Dynamic>
      <Show when={local.kicker}>
        {(kicker) => (
          <p class="font-heading text-xs uppercase tracking-[0.3em] text-grimorio-gold">{kicker()}</p>
        )}
      </Show>
    </div>
  )
}
