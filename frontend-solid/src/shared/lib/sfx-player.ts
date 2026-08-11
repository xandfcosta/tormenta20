/** UI sound cues. Kept tiny + synthesized (no audio assets → CSP/offline-safe,
 *  zero bytes shipped). `open`/`back` are the diegetic enter/exit pair (diving
 *  into a chronicle rises; popping back with Esc falls). */
export type SfxName = 'hover' | 'select' | 'transition' | 'open' | 'back'

/** The audio backend seam — wrapped so components never touch Web Audio and so
 *  tests can swap in a fake (project I/O-mock rule). */
export interface SfxPlayer {
  play(name: SfxName): void
}

type Blip = {
  type: OscillatorType
  from: number
  to: number
  dur: number
  gain: number
}

// Short, quiet enveloped tones — tuned to feel like soft parchment/iron cues,
// not arcade beeps. Descending sweep on transition reads as a "whoosh".
const CUES: Record<SfxName, Blip> = {
  hover: { type: 'triangle', from: 1300, to: 1100, dur: 0.05, gain: 0.03 },
  select: { type: 'sine', from: 520, to: 780, dur: 0.12, gain: 0.08 },
  transition: { type: 'sine', from: 620, to: 200, dur: 0.28, gain: 0.06 },
  // Enter/exit pair: open rises (diving in), back falls (popping out).
  open: { type: 'sine', from: 320, to: 760, dur: 0.22, gain: 0.07 },
  back: { type: 'triangle', from: 660, to: 240, dur: 0.18, gain: 0.06 },
}

/** Web Audio implementation. Every entry point is guarded so a missing/blocked
 *  AudioContext degrades to silence — sound is a nicety, never an app error. */
export class WebAudioSfxPlayer implements SfxPlayer {
  private ctx: AudioContext | null = null

  play(name: SfxName): void {
    try {
      const ctx = this.context()
      if (!ctx) return
      // Autoplay policy parks the context until a gesture; play() is only
      // ever called from a click/hover, so resuming here is allowed.
      if (ctx.state === 'suspended') void ctx.resume()
      blip(ctx, CUES[name])
    } catch {
      // swallow — audio must never break the UI
    }
  }

  private context(): AudioContext | null {
    if (this.ctx) return this.ctx
    const Ctor =
      window.AudioContext ??
      (window as unknown as { webkitAudioContext?: typeof AudioContext })
        .webkitAudioContext
    if (!Ctor) return null
    this.ctx = new Ctor()
    return this.ctx
  }
}

function blip(ctx: AudioContext, { type, from, to, dur, gain }: Blip): void {
  const t = ctx.currentTime
  const osc = ctx.createOscillator()
  const env = ctx.createGain()
  osc.type = type
  osc.frequency.setValueAtTime(from, t)
  osc.frequency.exponentialRampToValueAtTime(to, t + dur)
  // exponential ramps can't touch 0, so bracket with a tiny floor.
  env.gain.setValueAtTime(0.0001, t)
  env.gain.exponentialRampToValueAtTime(gain, t + 0.012)
  env.gain.exponentialRampToValueAtTime(0.0001, t + dur)
  osc.connect(env).connect(ctx.destination)
  osc.start(t)
  osc.stop(t + dur + 0.02)
}

/** Factory (the test seam — mock this module to fake the player). */
export function createSfxPlayer(): SfxPlayer {
  return new WebAudioSfxPlayer()
}
