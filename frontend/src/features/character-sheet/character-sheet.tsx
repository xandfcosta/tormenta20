import type { ReactNode } from 'react'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import { CharacterSheetDesktop } from './character-sheet-desktop'
import { CharacterSheetMobile } from './character-sheet-mobile'
import type { Character } from '@/shared/api/api'

// Re-exported so callers merging a control into the phone bar (session view)
// can match the tab styling without reaching into a layout internal.
export { BOTTOM_TAB } from './character-sheet-mobile'

/**
 * The full editable character sheet, self-contained around a `Character` so it
 * renders both on the standalone editor page and as the in-session player
 * surface. Picks a layout by viewport: a two-column desktop sheet, or a
 * one-block-at-a-time phone sheet with a bottom tab bar. `mobileBarSlot` lets
 * a caller (the session player view) merge an extra control into that bar.
 */
export function CharacterSheet({
  character,
  mobileBarSlot,
  inSession,
}: {
  character: Character
  mobileBarSlot?: ReactNode
  /** In a live session the Campanhas block is dropped — already in one. */
  inSession?: boolean
}) {
  const isDesktop = useMediaQuery('(min-width: 1024px)')
  if (isDesktop)
    return <CharacterSheetDesktop character={character} inSession={inSession} />
  return (
    <CharacterSheetMobile
      character={character}
      barSlot={mobileBarSlot}
      inSession={inSession}
    />
  )
}
