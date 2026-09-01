import { AudioGate } from './audio-gate'
import { type SfxName, playCue } from './sfx-cues'

export type { SfxName }

/** The audio backend seam — wrapped so components never touch Web Audio and so
 *  tests can swap in a fake (project I/O-mock rule). `volume` is 0–1. */
export interface SfxPlayer {
  play(name: SfxName, volume: number): void
}

/** Web Audio implementation. Every entry point is guarded so a missing/blocked
 *  AudioContext degrades to silence — sound is a nicety, never an app error. */
export class WebAudioSfxPlayer implements SfxPlayer {
  /** The gate comes in as a parameter: it owns the autoplay rule, and tests
   *  exercise that rule against it directly (`audio-gate.test.ts`). */
  constructor(private readonly gate: AudioGate<AudioContext> = browserAudioGate()) {}

  play(name: SfxName, volume: number): void {
    try {
      // Null until the player's first gesture built the context — scheduling
      // into a parked context is exactly how the cue got lost (ALE-165).
      const ctx = this.gate.ready()
      if (ctx) playCue(ctx, name, volume)
    } catch {
      // swallow — audio must never break the UI
    }
  }
}

/** Browser wiring: the shared context is born from the first gesture on `window`. */
function browserAudioGate(): AudioGate<AudioContext> {
  return new AudioGate(openAudioContext, window)
}

function openAudioContext(): AudioContext | null {
  const Ctor =
    window.AudioContext ??
    (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
  return Ctor ? new Ctor() : null
}

/** Factory (the test seam — mock this module to fake the player). */
export function createSfxPlayer(): SfxPlayer {
  return new WebAudioSfxPlayer()
}
