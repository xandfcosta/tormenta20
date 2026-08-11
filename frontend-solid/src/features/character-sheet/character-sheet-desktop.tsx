import { Dynamic } from 'solid-js/web'
import { For, Show } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { SHEET_PANELS, resolveSheetTab } from './sheet-sections'

export type CharacterSheetDesktopProps = {
  character: Character
  inSession?: boolean
  tab: string
  onTabChange: (value: string) => void
}

/**
 * Wide-viewport sheet: the selected block's content beside a labelled icon rail
 * (every block, always visible). Phones use the bottom-bar layout instead.
 *
 * The React version kept every visited block mounted (`forceMount` + a
 * `useVisitedTabs` ref) because a switch otherwise re-ran all of them. That
 * scaffolding is gone: here a switch only touches what reads the signal.
 */
export function CharacterSheetDesktop(props: CharacterSheetDesktopProps) {
  // The shared tab value may come from the phone set (e.g. "vitals"), which has
  // no desktop block — fall back to the first so content always shows.
  const active = () => {
    const resolved = resolveSheetTab(props.tab)
    return SHEET_PANELS.some((p) => p.value === resolved) ? resolved : SHEET_PANELS[0].value
  }

  return (
    <div
      data-sheet-root
      class={cn(
        'grid h-full min-h-0 grid-rows-[1fr_auto] gap-3 overflow-hidden',
        !props.inSession && 'p-3',
      )}
    >
      <Tabs
        value={active()}
        onChange={(value) => props.onTabChange(value)}
        orientation="vertical"
        class="flex min-h-0 items-stretch gap-3"
      >
        {/* Content is a dense editing FORM (selects, inline edits) — Tab, not
            arrow-roving, is the right keyboard here. Only the rail is a
            scene-nav region; PageUp/PageDown bump blocks, Esc leaves. */}
        <div class="min-h-0 min-w-0 flex-1">
          <For each={SHEET_PANELS}>
            {(section) => (
              <TabsContent
                value={section.value}
                class="m-0 flex h-full min-h-0 flex-col overflow-hidden"
              >
                <Dynamic component={section.component} character={props.character} />
              </TabsContent>
            )}
          </For>
        </div>

        <TabsList
          data-nav-region="rail"
          data-nav-layout="column"
          class="flex h-full shrink-0 flex-col gap-1 rounded-lg border bg-card p-1"
        >
          <For each={SHEET_PANELS}>
            {(section) => (
              <TabsTrigger
                value={section.value}
                aria-label={section.label}
                class={cn(
                  'relative w-32 flex-1 justify-start gap-2 px-2',
                  // Selected = gold, the app's selection language everywhere.
                  'data-[selected]:text-grimorio-gold',
                  // Irrelevant blocks (Magias for non-casters) stay reachable
                  // but stop competing for scan attention.
                  section.dim?.(props.character) && 'opacity-40',
                )}
              >
                <Dynamic component={section.icon} class="size-5 shrink-0" />
                <span class="truncate text-xs">{section.label}</span>
                <Show when={section.badge}>
                  {(badge) => (
                    <span class="absolute -right-1 -top-1">
                      <Dynamic component={badge()} character={props.character} />
                    </span>
                  )}
                </Show>
              </TabsTrigger>
            )}
          </For>
        </TabsList>
      </Tabs>
    </div>
  )
}
