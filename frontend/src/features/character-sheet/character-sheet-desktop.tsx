import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { SheetHeader } from './sheet-header'
import { VitalsAside } from './vitals-aside'
import { SHEET_PANELS } from './sheet-sections'
import type { Character } from '@/shared/api/api'

/**
 * Wide-viewport sheet: header spanning the top, then three columns — a slim
 * icon nav rail (the eight blocks, always visible), the persistent vitals
 * column, and the selected block's content. Phones use the bottom-bar layout
 * instead (see `character-sheet-mobile`), which mirrors the same icon set.
 */
export function CharacterSheetDesktop({
  character,
  inSession,
}: {
  character: Character
  inSession?: boolean
}) {
  const panels = inSession
    ? SHEET_PANELS.filter((p) => p.value !== 'campaigns')
    : SHEET_PANELS
  return (
    <div
      className={cn(
        'grid h-full min-h-0 grid-rows-[auto_1fr] gap-3 overflow-hidden',
        !inSession && 'p-3',
      )}
    >
      <SheetHeader character={character} />
      <Tabs
        defaultValue={panels[0]!.value}
        orientation="vertical"
        className="grid min-h-0 gap-3 lg:grid-cols-[minmax(18rem,22rem)_1fr_auto]"
      >
        <VitalsAside character={character} />

        <div className="min-h-0">
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

        <TabsList className="flex h-full flex-col gap-1 rounded-lg border bg-card p-1">
          {panels.map((s) => (
            <TabsTrigger
              key={s.value}
              value={s.value}
              aria-label={s.label}
              title={s.label}
              className="relative size-11 flex-none justify-center p-0"
            >
              <s.icon className="size-5" />
              {s.badge && (
                <span className="absolute -right-1 -top-1">
                  {s.badge(character)}
                </span>
              )}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>
    </div>
  )
}
