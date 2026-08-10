/**
 * HealthVignette — a "bleeding screen" cue driven by the character's HP, layered
 * over the sheet scene. Healthy = nothing (no distraction); below half HP a red
 * vignette creeps in and intensifies toward 0; at 0-or-below (dying) the whole
 * scene desaturates to grey. Purely presentational + pointer-events-none.
 */

export type VignetteState =
  | { kind: 'none' }
  | { kind: 'wound'; t: number } // t in (0,1): 0 near half HP, 1 near 0 HP
  | { kind: 'dying' }

/**
 * Pure severity from current/max HP, so the ramp is unit-testable.
 *
 * @example healthVignette(10, 40) // { kind: 'wound', t: 0.5 }
 */
export function healthVignette(current: number, max: number): VignetteState {
  if (current <= 0) return { kind: 'dying' }
  const ratio = max > 0 ? current / max : 1
  if (ratio >= 0.5) return { kind: 'none' }
  return { kind: 'wound', t: (0.5 - ratio) / 0.5 }
}

export function HealthVignette({
  current,
  max,
}: {
  current: number
  max: number
}) {
  const state = healthVignette(current, max)
  if (state.kind === 'none') return null

  if (state.kind === 'dying') {
    return (
      <div
        aria-hidden
        data-slot="health-vignette"
        data-state="dying"
        className="pointer-events-none absolute inset-0 z-20 backdrop-grayscale transition-opacity duration-700"
        style={{
          background:
            'radial-gradient(ellipse at center, oklch(0 0 0 / 0.2) 25%, oklch(0 0 0 / 0.8) 115%)',
        }}
      />
    )
  }

  const { t } = state
  return (
    <div
      aria-hidden
      data-slot="health-vignette"
      data-state="wound"
      className="pointer-events-none absolute inset-0 z-20 transition-opacity duration-500"
      style={{
        // Grows opaquer and the clear center shrinks as HP nears 0.
        opacity: 0.2 + t * 0.65,
        background: `radial-gradient(ellipse at center, transparent ${58 - t * 30}%, oklch(0.45 0.22 25 / 0.95) 115%)`,
      }}
    />
  )
}
