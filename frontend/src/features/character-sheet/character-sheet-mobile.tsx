import type { ReactNode } from 'react'
import { HeartPulse } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { sheetBg } from '@/shared/lib/sheet-theme'
import { SheetHeader } from './sheet-header'
import { VitalsAside } from './vitals-aside'
import { SHEET_PANELS, type SheetSection } from './sheet-sections'
import type { Character } from '@/shared/api/api'

// Vitais leads the mobile sections: header + vitals aside as one scroll block.
const VITALS_SECTION: SheetSection = {
  value: 'vitals',
  label: 'Vitais',
  icon: HeartPulse,
  render: (c) => (
    <div className="h-full space-y-3 overflow-y-auto">
      <SheetHeader character={c} />
      <VitalsAside character={c} />
    </div>
  ),
}

// A bottom-bar cell: minimalist icon-only, sharing the width equally (flex-1)
// so all sections fit without a horizontal scroll — which lets the bar avoid
// `overflow` entirely, so an over-icon badge never gets clipped. Active = amber
// icon + a short amber marker along the top edge.
export const BOTTOM_TAB =
  'group/tab relative flex h-full flex-1 items-center justify-center rounded-none border-0 px-1 text-muted-foreground transition-colors data-[state=active]:bg-transparent data-[state=active]:text-[color:var(--primary)] data-[state=active]:shadow-none data-[state=active]:before:absolute data-[state=active]:before:left-1/2 data-[state=active]:before:top-0 data-[state=active]:before:h-0.5 data-[state=active]:before:w-6 data-[state=active]:before:-translate-x-1/2 data-[state=active]:before:rounded-full data-[state=active]:before:bg-[color:var(--primary)]'

/**
 * Phone sheet: one block at a time over a bottom tab bar (thumb-reach), instead
 * of the desktop two-column layout whose top tab strip overflows and is awkward
 * to tap. Minimalist icon-only tabs share the width; the active one is amber
 * with a short top marker. `barSlot` merges an extra control (the in-session
 * rail) into the same bar so a phone never stacks two bottom bars. In a session
 * the Campanhas block is dropped — you're already inside one.
 */
export function CharacterSheetMobile({
  character,
  barSlot,
  inSession,
}: {
  character: Character
  barSlot?: ReactNode
  inSession?: boolean
}) {
  const panels = inSession
    ? SHEET_PANELS.filter((p) => p.value !== 'campaigns')
    : SHEET_PANELS
  const sections = [VITALS_SECTION, ...panels]

  return (
    <Tabs
      defaultValue={VITALS_SECTION.value}
      className={cn('flex h-full min-h-0 w-full min-w-0 flex-col', sheetBg)}
    >
      <div className="min-h-0 min-w-0 flex-1 overflow-hidden">
        {sections.map((s) => (
          <TabsContent
            key={s.value}
            value={s.value}
            // Bounded height (no outer scroll) so each panel's own pinned
            // header + inner scroll region take over — the header stays put
            // while the table grows and scrolls. Blocky sections (Vitais)
            // wrap their own overflow-y-auto.
            className="flex h-full min-h-0 flex-col overflow-hidden p-2"
          >
            {s.render(character)}
          </TabsContent>
        ))}
      </div>
      {/* Full-width bar; cells share the width (no scroll, no overflow) so the
          over-icon badge is never clipped. */}
      <div className="w-full shrink-0 border-t border-border/60 bg-card/95 backdrop-blur">
        <TabsList className="flex h-14 w-full items-stretch gap-0 rounded-none border-0 bg-transparent p-0">
          {sections.map((s) => (
            <TabsTrigger
              key={s.value}
              value={s.value}
              aria-label={s.label}
              title={s.label}
              className={BOTTOM_TAB}
            >
              <span className="relative flex items-center justify-center">
                <s.icon className="size-5 shrink-0" />
                {s.badge && (
                  <span className="absolute -right-2 -top-1.5">
                    {s.badge(character)}
                  </span>
                )}
              </span>
            </TabsTrigger>
          ))}
          {barSlot}
        </TabsList>
      </div>
    </Tabs>
  )
}
