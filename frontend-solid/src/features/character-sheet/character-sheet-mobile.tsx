import { Dynamic } from 'solid-js/web'
import { For, type JSX, Show } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { CharacterHud } from './character-hud'
import { SHEET_PANELS, resolveSheetTab } from './sheet-sections'

/**
 * A bottom-bar cell: icon-only, sharing the width equally so every block fits
 * without a horizontal scroll — which lets the bar avoid `overflow` entirely,
 * so an over-icon badge is never clipped. Active = gold icon + a short marker
 * along the top edge.
 *
 * Exported so a caller merging a control into the same bar (the in-session
 * player view) matches the styling without reaching into a layout internal.
 */
export const BOTTOM_TAB =
  'group/tab relative flex h-full flex-1 items-center justify-center rounded-none border-0 px-1 text-muted-foreground transition-colors data-[selected]:bg-transparent data-[selected]:text-grimorio-gold data-[selected]:shadow-none data-[selected]:before:absolute data-[selected]:before:left-1/2 data-[selected]:before:top-0 data-[selected]:before:h-0.5 data-[selected]:before:w-6 data-[selected]:before:-translate-x-1/2 data-[selected]:before:rounded-full data-[selected]:before:bg-grimorio-gold'

export type CharacterSheetMobileProps = {
  character: Character
  /** Merged into the bottom bar so a phone never stacks two bars. */
  barSlot?: JSX.Element
  inSession?: boolean
  tab: string
  onTabChange: (value: string) => void
}

/**
 * Phone sheet: one block at a time over a bottom tab bar (thumb reach), instead
 * of the desktop two-column layout whose rail would overflow and be awkward to
 * tap.
 */
export function CharacterSheetMobile(props: CharacterSheetMobileProps) {
  const active = () => {
    const resolved = resolveSheetTab(props.tab)
    return SHEET_PANELS.some((s) => s.value === resolved) ? resolved : SHEET_PANELS[0].value
  }

  return (
    <Tabs
      value={active()}
      onChange={(value) => props.onTabChange(value)}
      class="flex h-full min-h-0 w-full min-w-0 flex-col"
    >
      {/* min-h-40 floor: on a short viewport (phone landscape) the shrink-0 bar
          would squeeze this flex-1 area to 0px and the active block's header
          would vanish under the top nav. With a floor the shell scrolls. */}
      <div class="min-h-40 min-w-0 flex-1 overflow-hidden">
        <For each={SHEET_PANELS}>
          {(section) => (
            <TabsContent
              value={section.value}
              class="flex h-full min-h-0 flex-col overflow-hidden p-2"
            >
              <Dynamic component={section.component} character={props.character} />
            </TabsContent>
          )}
        </For>
      </div>

      {/* Above the tab bar, below the block: PV/PM stay reachable from every
          block — at the table they are what the player touches most. */}
      <CharacterHud character={props.character} class="shrink-0" />

      <div class="w-full shrink-0 border-t border-border/60 bg-card/95 backdrop-blur">
        <TabsList class="flex h-14 w-full items-stretch gap-0 rounded-none border-0 bg-transparent p-0 landscape:h-11">
          <For each={SHEET_PANELS}>
            {(section) => (
              <TabsTrigger
                value={section.value}
                aria-label={section.label}
                title={section.label}
                class={cn(BOTTOM_TAB, section.dim?.(props.character) && 'opacity-40')}
              >
                <span class="relative flex items-center justify-center">
                  <Dynamic component={section.icon} class="size-5 shrink-0" />
                  <Show when={section.badge}>
                    {(badge) => (
                      <span class="absolute -right-2 -top-1.5">
                        <Dynamic component={badge()} character={props.character} />
                      </span>
                    )}
                  </Show>
                </span>
              </TabsTrigger>
            )}
          </For>
          {props.barSlot}
        </TabsList>
      </div>
    </Tabs>
  )
}
