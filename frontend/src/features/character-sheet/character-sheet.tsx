import { SheetSearch } from './sheet-search'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { type ReactNode } from 'react'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import { useSfx } from '@/shared/lib/use-sfx'
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
  // Owned here (above the viewport switch) so the selected block survives a
  // desktop↔mobile swap; each layout falls back to its own first tab when the
  // shared value isn't in its set (e.g. mobile-only "Vitais"). Persisted in
  // the URL: a phone-tab eviction or accidental nav no longer loses context.
  const navigate = useNavigate()
  const sfx = useSfx()
  const search = useSearch({ strict: false }) as { tab?: string }
  const tab = search.tab ?? ''
  const setTab = (next: string) => {
    if (next !== tab) sfx('select')
    navigate({
      to: '.',
      search: (prev: Record<string, unknown>) => ({ ...prev, tab: next }),
      replace: true,
    })
  }
  const layout = isDesktop ? (
    <CharacterSheetDesktop
      character={character}
      inSession={inSession}
      tab={tab}
      onTabChange={setTab}
    />
  ) : (
    <CharacterSheetMobile
      character={character}
      barSlot={mobileBarSlot}
      inSession={inSession}
      tab={tab}
      onTabChange={setTab}
    />
  )
  return (
    <>
      {layout}
      <SheetSearch character={character} onNavigate={setTab} />
    </>
  )
}
