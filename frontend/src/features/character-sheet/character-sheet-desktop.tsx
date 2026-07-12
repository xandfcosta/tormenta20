import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { sheetBg } from '@/shared/lib/sheet-theme'
import { SheetHeader } from './sheet-header'
import { VitalsAside } from './vitals-aside'
import { SHEET_PANELS } from './sheet-sections'
import type { Character } from '@/shared/api/api'

/**
 * Wide-viewport sheet: header spanning the top, vitals aside pinned on the
 * left, and the eight blocks as a right-panel tab strip. Phones use the
 * bottom-bar layout instead (see `character-sheet-mobile`).
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
        'grid h-full grid-rows-[auto_1fr] gap-3 overflow-hidden p-3 lg:grid-cols-[minmax(300px,26rem)_1fr]',
        sheetBg,
      )}
    >
      <SheetHeader character={character} className="lg:col-span-2" />
      <VitalsAside character={character} />
      <Tabs
        defaultValue={panels[0]!.value}
        className="flex min-h-0 flex-col gap-2 overflow-hidden"
      >
        <TabsList className="max-w-full self-start overflow-x-auto [&>button]:shrink-0">
          {panels.map((s) => (
            <TabsTrigger key={s.value} value={s.value} className="gap-1.5">
              <s.icon className="size-3.5" /> {s.label}
              {s.badge?.(character)}
            </TabsTrigger>
          ))}
        </TabsList>
        {panels.map((s) => (
          <TabsContent
            key={s.value}
            value={s.value}
            className="flex min-h-0 flex-1 flex-col overflow-hidden"
          >
            {s.render(character)}
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
