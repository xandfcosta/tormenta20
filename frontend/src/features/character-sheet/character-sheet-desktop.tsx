import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/shared/ui/tooltip'
import { cn } from '@/shared/lib/utils'
import { CharacterHud } from './character-hud'
import { SHEET_PANELS } from './sheet-sections'
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
  const panels = inSession
    ? SHEET_PANELS.filter((p) => p.value !== 'campaigns')
    : SHEET_PANELS
  // The shared tab value may come from the mobile set (e.g. "vitals"), which
  // has no desktop panel — fall back to the first block so content shows.
  const active = panels.some((p) => p.value === tab) ? tab : panels[0]!.value
  const activeLabel = panels.find((p) => p.value === active)?.label
  return (
    <div
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
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          {/* Persistent label for the active block — the icon rail alone
              doesn't say which tab you're on without hovering. */}
          <h2 className="shrink-0 text-sm font-bold tracking-tight">
            {activeLabel}
          </h2>
          <div className="min-h-0 flex-1">
            {panels.map((s) => (
              <TabsContent
                key={s.value}
                value={s.value}
                className="m-0 flex h-full min-h-0 flex-col overflow-hidden"
              >
                {s.render(character)}
              </TabsContent>
            ))}
          </div>
        </div>

        <TooltipProvider delayDuration={200}>
          <TabsList className="flex h-full shrink-0 flex-col gap-1 rounded-lg border bg-card p-1">
            {panels.map((s) => (
              <Tooltip key={s.value}>
                <TooltipTrigger asChild>
                  <TabsTrigger
                    value={s.value}
                    aria-label={s.label}
                    className={cn(
                      'relative w-11 flex-1 justify-center p-0 xl:w-32 xl:justify-start xl:gap-2 xl:px-2',
                      // Irrelevant tabs (Magias for non-casters) stay reachable
                      // but stop competing for scan attention.
                      s.dim?.(character) && 'opacity-40',
                    )}
                  >
                    <s.icon className="size-5 shrink-0" />
                    <span className="hidden truncate text-xs xl:inline">
                      {s.label}
                    </span>
                    {s.badge && (
                      <span className="absolute -right-1 -top-1">
                        {s.badge(character)}
                      </span>
                    )}
                  </TabsTrigger>
                </TooltipTrigger>
                <TooltipContent side="left">{s.label}</TooltipContent>
              </Tooltip>
            ))}
          </TabsList>
        </TooltipProvider>
      </Tabs>
      <CharacterHud character={character} className="rounded-xl" />
    </div>
  )
}
