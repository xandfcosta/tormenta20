import { useCallback } from 'react'
import { useUiStore } from '@/shared/stores/ui-store'
import { createSfxPlayer, type SfxName, type SfxPlayer } from './sfx-player'

// One AudioContext for the whole app, created lazily on first real play (i.e.
// after a user gesture, so the autoplay policy is satisfied).
let shared: SfxPlayer | null = null
function player(): SfxPlayer {
  if (!shared) shared = createSfxPlayer()
  return shared
}

/**
 * Returns `play(name)` for the UI sound cues. It fires only when the user has
 * enabled sound (persisted in ui-store, off by default), so the gate lives in
 * one place and callers can wire it unconditionally.
 *
 * @example
 *   const sfx = useSfx()
 *   <button onClick={() => { sfx('select'); go() }} onMouseEnter={() => sfx('hover')} />
 */
export function useSfx(): (name: SfxName) => void {
  const enabled = useUiStore((s) => s.sfx)
  return useCallback(
    (name: SfxName) => {
      if (enabled) player().play(name)
    },
    [enabled],
  )
}
