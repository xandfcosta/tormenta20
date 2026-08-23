import { Dynamic } from 'solid-js/web'
import { For, Show } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { cn } from '@/shared/lib/utils'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { CharacterHud } from './character-hud'
import { SHEET_PANELS, resolveSheetTab } from './sheet-sections'

export type CharacterSheetDesktopProps = {
  character: Character
  inSession?: boolean
  /** Sem o HUD: quem monta já mostra o cartão de combate acima (ALE-122). */
  hudless?: boolean
  /** Ver `CharacterSheetProps.glance` — a ficha do mestre no painel. */
  glance?: boolean
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
        // Sem o HUD a segunda faixa do grid deixa de existir, senão sobra uma
        // linha vazia embaixo do conteúdo (ALE-122).
        'grid h-full min-h-0 gap-3 overflow-hidden',
        props.hudless ? 'grid-rows-[1fr]' : 'grid-rows-[1fr_auto]',
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
                <Dynamic
                component={section.component}
                character={props.character}
                glance={props.glance}
                inSession={props.inSession}
              />
              </TabsContent>
            )}
          </For>
        </div>

        <TabsList
          data-nav-region="rail"
          data-nav-layout="column"
          class="flex h-full shrink-0 flex-col gap-1 rounded-md border bg-card p-1"
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

      {/* The `auto` row of the grid: the HUD is chrome every block sits above,
          not a block of its own. */}
      <Show when={!props.hudless}>
        <CharacterHud character={props.character} class="rounded-none" />
      </Show>
    </div>
  )
}
