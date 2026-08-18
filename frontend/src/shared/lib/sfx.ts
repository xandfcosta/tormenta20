import type { UiStore } from '@/shared/stores/ui-store'
import { type SfxName, type SfxPlayer, createSfxPlayer } from './sfx-player'

/** What a scene calls to fire a cue. */
export type PlayCue = (name: SfxName) => void

// One AudioContext for the whole app, guarded by the gate inside the player:
// the context itself is only born on the first user gesture.
let shared: SfxPlayer | null = null
function defaultPlayer(): SfxPlayer {
  if (!shared) shared = createSfxPlayer()
  return shared
}

/** Touch-first devices fire `hover` on the very tap that fires `select`, so a
 *  single press is heard twice. Read per call and not cached: a tablet gains
 *  and loses a mouse under the same page. */
function coarsePointer(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia('(pointer: coarse)').matches
}

/**
 * Returns `play(name)` for the UI sound cues, gated on the user having enabled
 * sound (persisted in the UI store, off by default) — so the gate lives in one
 * place and callers wire it unconditionally.
 *
 * The store and player both come in as parameters: no module-level `useUiStore`
 * import, and tests pass a fake player instead of mocking Web Audio.
 *
 * @example const sfx = createSfx(ui); <button onClick={() => sfx('select')} />
 */
export function createSfx(
  ui: UiStore,
  player: () => SfxPlayer = defaultPlayer,
  isCoarsePointer: () => boolean = coarsePointer,
): PlayCue {
  // Resolved EAGERLY, at scene setup, instead of inside the callback: building
  // the player is what subscribes the gesture listener that unlocks audio, and
  // it has to be listening BEFORE the player's first press — that press is the
  // one moment the browser lets the AudioContext be born running (ALE-165).
  const audio = player()
  return (name: SfxName): void => {
    if (!ui.sfx()) return
    if (name === 'hover' && isCoarsePointer()) return
    audio.play(name)
  }
}

/**
 * The sound switch, for every screen that offers one. Turning sound ON answers
 * with a cue: the click is the very gesture that unlocks the audio, so it is
 * the first instant a confirmation can be heard — and silence right after
 * flipping the switch is what made people think sound was broken (ALE-165).
 *
 * @example <button onClick={createSfxToggle(ui, sfx)}>Som</button>
 */
export function createSfxToggle(ui: UiStore, sfx: PlayCue): () => void {
  return () => {
    ui.toggleSfx()
    if (ui.sfx()) sfx('select')
  }
}
