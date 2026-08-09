import type * as React from 'react'
import { cn } from '@/shared/lib/utils'

/**
 * SceneTitle — a cinematic title for the Grimório scenes: Cinzel caps, wide
 * tracking, and a soft arcane glow. An optional `kicker` renders a small gold
 * eyebrow beneath it. Meant to be used inside a `.scene-grimorio` scope.
 *
 * @example
 * <SceneTitle kicker="— Grimório de Arton —">Tormenta 20</SceneTitle>
 */
type SceneTitleProps = React.ComponentProps<'h1'> & {
  as?: 'h1' | 'h2'
  kicker?: React.ReactNode
}

function SceneTitle({
  className,
  children,
  kicker,
  as: Tag = 'h1',
  ...props
}: SceneTitleProps) {
  return (
    <div
      data-slot="scene-title"
      className="flex flex-col items-center gap-1 text-center"
    >
      <Tag
        className={cn(
          'scene-title-glow font-heading font-bold text-[clamp(2.2rem,8vw,3.4rem)] uppercase leading-tight tracking-[0.18em] text-foreground',
          className,
        )}
        {...props}
      >
        {children}
      </Tag>
      {kicker ? (
        <p className="font-heading text-xs uppercase tracking-[0.3em] text-grimorio-gold">
          {kicker}
        </p>
      ) : null}
    </div>
  )
}

export { SceneTitle }
