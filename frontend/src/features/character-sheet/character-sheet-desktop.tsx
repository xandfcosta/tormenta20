import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { CharacterHud } from './character-hud'
import { SHEET_PANELS, resolveSheetTab } from './sheet-sections'
import { useVisitedTabs } from './use-visited-tabs'
import type { Character } from '@/shared/api/api'

/**
 * Wide-viewport sheet: the selected block's content beside a slim icon nav
 * rail (the eight blocks, always visible), over a game-style HUD pinned to the
 * bottom. The HUD carries identity, PV/PM and the attribute/combat/magic stats
 * (the old aside column), so the content spans the full width. Phones use the
 * bottom-bar layout instead (see `character-sheet-mobile`).
 */
export function CharacterSheetDesktop({
  character,
  inSession,
  tab,
  onTabChange,
}: {
  character: Character
  inSession?: boolean
  tab: string
  onTabChange: (value: string) => void
}) {
  const panels = SHEET_PANELS
  // The shared tab value may come from the mobile set (e.g. "vitals"), which
  // has no desktop panel — fall back to the first block so content shows.
  const resolved = resolveSheetTab(tab)
  const active = panels.some((p) => p.value === resolved)
    ? resolved
    : panels[0]!.value
  // Keep every already-opened block mounted so a revisit toggles visibility
  // instead of re-mounting its (heavy) content — the tab-switch jank fix.
  const visited = useVisitedTabs(active)
  return (
    <div
      data-sheet-root
      className={cn(
        'grid h-full min-h-0 grid-rows-[1fr_auto] gap-3 overflow-hidden',
        !inSession && 'p-3',
      )}
    >
      <Tabs
        value={active}
        onValueChange={onTabChange}
        orientation="vertical"
        className="flex min-h-0 items-stretch gap-3"
      >
        {/* Content is a dense editing FORM (selects, inline edits) — Tab, not
            arrow-roving, is the right keyboard here. Only the block rail is a
            scene-nav region; PageUp/PageDown bump blocks, Esc leaves. */}
        <div className="min-h-0 min-w-0 flex-1">
          {panels.map((s) => (
            <TabsContent
              key={s.value}
              value={s.value}
              // Once visited, stay mounted (forceMount) and just hide when
              // inactive — the `data-[state=inactive]:hidden` variant outranks
              // `flex` so the hidden panel is truly display:none.
              forceMount={visited.has(s.value) || undefined}
              className="m-0 flex h-full min-h-0 flex-col overflow-hidden data-[state=inactive]:hidden"
            >
              {s.render(character)}
            </TabsContent>
          ))}
        </div>

        {/* Labeled rail at every desktop width — the panel already carries
            its own big title, so no floating duplicate above the content. */}
        <TabsList
          data-nav-region="rail"
          data-nav-layout="column"
          className="flex h-full shrink-0 flex-col gap-1 rounded-lg border bg-card p-1"
        >
          {panels.map((s) => (
            <TabsTrigger
              key={s.value}
              value={s.value}
              aria-label={s.label}
              className={cn(
                'relative w-32 flex-1 justify-start gap-2 px-2',
                // Selected block = gold (the app's "selected = gold" language);
                // text+icon inherit currentColor, the indicator bar (`after`)
                // is tinted too.
                'data-[state=active]:text-grimorio-gold data-[state=active]:after:bg-grimorio-gold',
                // Irrelevant tabs (Magias for non-casters) stay reachable
                // but stop competing for scan attention.
                s.dim?.(character) && 'opacity-40',
              )}
            >
              <s.icon className="size-5 shrink-0" />
              <span className="truncate text-xs">{s.label}</span>
              {s.badge && (
                <span className="absolute -right-1 -top-1">
                  {s.badge(character)}
                </span>
              )}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
      <CharacterHud character={character} className="rounded-xl" />
    </div>
  )
}
