import type { UiStore } from '@/shared/stores/ui-store'
import { type SfxName, type SfxPlayer, createSfxPlayer } from './sfx-player'

// One AudioContext for the whole app, created lazily on first real play (i.e.
// after a user gesture, so the autoplay policy is satisfied).
let shared: SfxPlayer | null = null
function defaultPlayer(): SfxPlayer {
  if (!shared) shared = createSfxPlayer()
  return shared
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
export function createSfx(ui: UiStore, player: () => SfxPlayer = defaultPlayer) {
  return (name: SfxName): void => {
    if (ui.sfx()) player().play(name)
  }
}
