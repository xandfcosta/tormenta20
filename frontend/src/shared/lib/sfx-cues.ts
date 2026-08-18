/** UI sound cues. Kept tiny + synthesized (no audio assets → CSP/offline-safe,
 *  zero bytes shipped). `open`/`back` are the diegetic enter/exit pair (diving
 *  into a chronicle rises; popping back with Esc falls); `turn` is the only
 *  one that carries INFORMATION and not decoration (ALE-180). */
export type SfxName = 'hover' | 'select' | 'transition' | 'open' | 'back' | 'turn'

/** A single oscillator gliding between two pitches. */
type Sweep = {
  shape: 'sweep'
  type: OscillatorType
  from: number
  to: number
  dur: number
  gain: number
}

/** Several partials struck together — fast attack, long tail. */
type Bell = {
  shape: 'bell'
  partials: readonly number[]
  dur: number
  gain: number
}

type Cue = Sweep | Bell

// Short, quiet enveloped tones — tuned to feel like soft parchment/iron cues,
// not arcade beeps. Descending sweep on transition reads as a "whoosh".
const CUES: Record<SfxName, Cue> = {
  hover: { shape: 'sweep', type: 'triangle', from: 1300, to: 1100, dur: 0.05, gain: 0.03 },
  select: { shape: 'sweep', type: 'sine', from: 520, to: 780, dur: 0.12, gain: 0.08 },
  transition: { shape: 'sweep', type: 'sine', from: 620, to: 200, dur: 0.28, gain: 0.06 },
  // Enter/exit pair: open rises (diving in), back falls (popping out).
  open: { shape: 'sweep', type: 'sine', from: 320, to: 760, dur: 0.22, gain: 0.07 },
  back: { shape: 'sweep', type: 'triangle', from: 660, to: 240, dur: 0.18, gain: 0.06 },
  // "Sua vez": a bronze bell, and the LOUDEST + LONGEST of the palette on
  // purpose. It is the only cue whose job is to reach someone who is looking at
  // the dice, at the GM, at their own phone — so it has to survive a table
  // talking over it. Everything else here is juice and stays out of the way.
  turn: { shape: 'bell', partials: [880, 1320], dur: 0.9, gain: 0.1 },
}

/**
 * Renders one cue into a RUNNING context. `volume` is 0–1 and multiplies the
 * cue's own gain, so the palette stays balanced while the player scales it.
 *
 * @example playCue(ctx, 'turn', 0.7)
 */
export function playCue(ctx: AudioContext, name: SfxName, volume: number): void {
  if (volume <= 0) return
  const cue = CUES[name]
  if (cue.shape === 'bell') {
    bell(ctx, cue, volume)
    return
  }
  sweep(ctx, cue, volume)
}

function sweep(ctx: AudioContext, { type, from, to, dur, gain }: Sweep, volume: number): void {
  const t = ctx.currentTime
  const osc = ctx.createOscillator()
  osc.type = type
  osc.frequency.setValueAtTime(from, t)
  osc.frequency.exponentialRampToValueAtTime(to, t + dur)
  osc.connect(envelope(ctx, gain * volume, t, 0.012, dur)).connect(ctx.destination)
  osc.start(t)
  osc.stop(t + dur + 0.02)
}

function bell(ctx: AudioContext, { partials, dur, gain }: Bell, volume: number): void {
  const t = ctx.currentTime
  partials.forEach((hz, index) => {
    // The upper partials of bronze are quieter and die FIRST — that decay
    // ratio is what reads as a bell instead of two beeps held together.
    const peak = gain * volume * (index === 0 ? 1 : 0.45)
    const tail = dur * (index === 0 ? 1 : 0.55)
    const osc = ctx.createOscillator()
    osc.type = 'sine'
    osc.frequency.setValueAtTime(hz, t)
    osc.connect(envelope(ctx, peak, t, 0.005, tail)).connect(ctx.destination)
    osc.start(t)
    osc.stop(t + tail + 0.02)
  })
}

/** Attack to `peak`, then decay to silence. Exponential ramps can't touch 0,
 *  so both ends are bracketed with a tiny floor. */
function envelope(ctx: AudioContext, peak: number, t: number, attack: number, dur: number): GainNode {
  const env = ctx.createGain()
  env.gain.setValueAtTime(0.0001, t)
  env.gain.exponentialRampToValueAtTime(Math.max(peak, 0.0002), t + attack)
  env.gain.exponentialRampToValueAtTime(0.0001, t + dur)
  return env
}
