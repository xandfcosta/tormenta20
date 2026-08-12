import { Show, type JSX } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { createMediaQuery } from '@/shared/lib/media-query'
import { CharacterSheetDesktop } from './character-sheet-desktop'
import { CharacterSheetMobile } from './character-sheet-mobile'

export { BOTTOM_TAB } from './character-sheet-mobile'

export type CharacterSheetProps = {
  character: Character
  /** Merged into the phone bar by the in-session player view. */
  mobileBarSlot?: JSX.Element
  /** In a live session the sheet blends into the session chrome. */
  inSession?: boolean
  tab: string
  onTabChange: (value: string) => void
}

/**
 * The full character sheet, self-contained around a `Character` so it renders
 * both on the standalone page and as the in-session player surface. Picks a
 * layout by viewport: a two-column desktop sheet, or a one-block-at-a-time
 * phone sheet with a bottom tab bar.
 *
 * The tab is owned by the CALLER (the page keeps it in the URL). The React
 * version could not do that: its comment explains that the router's state is a
 * `useSyncExternalStore` React cannot defer, so every switch ran as one
 * synchronous ~120ms task. It kept a `useState` mirror for the instant paint,
 * an effect to adopt the URL, and a second effect debouncing the URL write by
 * 250ms — the tab lived in two places and the URL lagged a quarter second
 * behind the screen. Here there is no re-render to schedule, so the URL is the
 * only home the value needs (same conclusion as ALE-72's campaign tabs).
 */
export function CharacterSheet(props: CharacterSheetProps) {
  const isDesktop = createMediaQuery('(min-width: 1024px)')

  return (
    <Show
      when={isDesktop()}
      fallback={
        <CharacterSheetMobile
          character={props.character}
          barSlot={props.mobileBarSlot}
          inSession={props.inSession}
          tab={props.tab}
          onTabChange={props.onTabChange}
        />
      }
    >
      <CharacterSheetDesktop
        character={props.character}
        inSession={props.inSession}
        tab={props.tab}
        onTabChange={props.onTabChange}
      />
    </Show>
  )
}
