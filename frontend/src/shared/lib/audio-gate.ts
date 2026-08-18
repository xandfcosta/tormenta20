/** The slice of `AudioContext` the gate touches. Narrow on purpose: the test
 *  fake is then two members instead of the whole Web Audio surface. */
export type GatedAudioContext = {
  readonly state: AudioContextState
  resume(): Promise<void>
}

/** Where the first user gesture is heard — `window` in the app, a fake in the
 *  tests (project rule: browser I/O comes in as a parameter). */
export type GestureTarget = {
  addEventListener(type: string, listener: () => void): void
}

/**
 * `pointerdown` and not `click`: it is the FIRST event of a press, so the
 * context is already running by the time the click handler asks for its cue.
 * `keydown` is here because the scenes are fully keyboard-driven (scene-nav) —
 * a player who never touches a pointer would otherwise never unlock sound.
 */
const ARMING_GESTURES = ['pointerdown', 'keydown'] as const

/**
 * Owns the app's single AudioContext and the browser's autoplay rule.
 *
 * A context built outside a user gesture is born `suspended`, and there
 * `currentTime` stays frozen at 0: the whole envelope gets scheduled in the
 * past and the cue dies in silence. That is why the first sounds of a session
 * never played — the first one the player actually heard was the second
 * scene's (ALE-165).
 *
 * So the context is not built at all until a gesture arrives, and it is built
 * INSIDE the handler, the one moment the browser lets it start `running`.
 * Before that `ready()` is null and cues are dropped on purpose: that is the
 * browser's ceiling, not a regression.
 *
 * @example const gate = new AudioGate(() => new AudioContext(), window)
 *          const ctx = gate.ready(); if (ctx) blip(ctx)
 */
export class AudioGate<T extends GatedAudioContext> {
  private ctx: T | null = null

  constructor(
    private readonly openContext: () => T | null,
    gestures: GestureTarget,
  ) {
    // Subscribed for good, not `once`: mobile browsers park the context when
    // the tab goes to the background, and the next gesture has to revive it.
    for (const gesture of ARMING_GESTURES) {
      gestures.addEventListener(gesture, () => this.arm())
    }
  }

  /** The context when a cue can actually be heard, `null` otherwise. */
  ready(): T | null {
    const ctx = this.ctx
    return ctx && ctx.state === 'running' ? ctx : null
  }

  /** Runs inside the gesture handler — see the class doc for why that matters. */
  private arm(): void {
    if (!this.ctx) this.ctx = this.openContext()
    if (this.ctx?.state === 'suspended') void this.ctx.resume()
  }
}
