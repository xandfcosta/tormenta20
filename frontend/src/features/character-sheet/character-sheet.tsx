import { SheetSearch } from './sheet-search'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { type ReactNode, useEffect, useState } from 'react'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import { useSceneNav } from '@/shared/lib/use-scene-nav'
import { useSfx } from '@/shared/lib/use-sfx'
import { CharacterSheetDesktop } from './character-sheet-desktop'
import { CharacterSheetMobile } from './character-sheet-mobile'
import { SHEET_PANELS, resolveSheetTab } from './sheet-sections'
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
  const urlTab = search.tab ?? ''
  // Local tab state drives the switch INSTANTLY. The tab used to live only in
  // the URL, so every switch was a router navigation — and TanStack's state is
  // a `useSyncExternalStore`, whose updates React can't defer, so the switch
  // ran as one synchronous ~120ms task even after the panels/HUD were memoized.
  // Now a click flips local state (a cheap, memoized re-render → instant paint)
  // and the URL is reconciled in a passive effect AFTER paint, off the switch's
  // critical path. The URL stays the source for deep-links/back (adopted below),
  // so nothing about that behaviour is lost.
  const [tab, setTab] = useState(urlTab)
  // Adopt external URL changes (deep-link, back/forward) into the local tab.
  useEffect(() => {
    setTab((cur) => (cur === urlTab ? cur : urlTab))
  }, [urlTab])
  // Reconcile local → URL, but OFF the switch's critical path. TanStack's
  // router state is a `useSyncExternalStore`, so a navigate() commits
  // synchronously and would drag a router re-render into the same task as the
  // click — re-inflating the blocking switch. Deferring it past a macrotask
  // lets the local switch paint first; debouncing collapses a fast burst of
  // switches into a single history write while keeping deep-links/back working.
  useEffect(() => {
    if (urlTab === tab) return
    const id = setTimeout(() => {
      navigate({
        to: '.',
        search: (prev: Record<string, unknown>) => ({ ...prev, tab }),
        replace: true,
      })
    }, 250)
    return () => clearTimeout(id)
  }, [tab, urlTab, navigate])
  const changeTab = (next: string) => {
    if (next === tab) return
    sfx('select')
    setTab(next)
  }
  // Shared scene-nav grammar (desktop only): arrows rove the block content and
  // cross to the icon rail; PageUp/PageDown are the block bumpers; Esc leaves to
  // the roster (never mid-session). Bumpers switch through local state only —
  // never navigate() — so the block swap stays instant (same reason as above).
  const cycleBlock = (delta: number) => {
    const vals = SHEET_PANELS.map((p) => p.value)
    const i = Math.max(0, vals.indexOf(resolveSheetTab(tab)))
    setTab(vals[(i + delta + vals.length) % vals.length])
  }
  useSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-sheet-root]'),
    onEscape: () => {
      if (!inSession) navigate({ to: '/characters' })
    },
    bumpers: { prev: () => cycleBlock(-1), next: () => cycleBlock(1) },
    sfx,
    active: isDesktop,
  })
  const layout = isDesktop ? (
    <CharacterSheetDesktop
      character={character}
      inSession={inSession}
      tab={tab}
      onTabChange={changeTab}
    />
  ) : (
    <CharacterSheetMobile
      character={character}
      barSlot={mobileBarSlot}
      inSession={inSession}
      tab={tab}
      onTabChange={changeTab}
    />
  )
  return (
    <>
      {layout}
      <SheetSearch character={character} onNavigate={changeTab} />
    </>
  )
}
