import type { ReactNode } from 'react'
import { HeartPulse } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/shared/ui/tabs'
import { cn } from '@/shared/lib/utils'
import { sheetBg } from '@/shared/lib/sheet-theme'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import { CharacterHud } from './character-hud'
import { VitalsAside } from './vitals-aside'
import { SHEET_PANELS, resolveSheetTab, type SheetSection } from './sheet-sections'
import type { Character } from '@/shared/api/api'

// Vitais leads the phone sections. Identity + PV/PM live in the persistent
// bottom HUD now, so this block is just the attributes/combat/magic column.
// `flex-1` fills the tab area so the card surface reaches the HUD instead of
// leaving a dead page-background gap under the stats (audit task 12).
function vitalsSection(): SheetSection {
  return {
    value: 'vitals',
    label: 'Vitais',
    icon: HeartPulse,
    render: (c) => <VitalsAside character={c} className="min-h-0 flex-1" />,
  }
}

// A bottom-bar cell: minimalist icon-only, sharing the width equally (flex-1)
// so all sections fit without a horizontal scroll — which lets the bar avoid
// `overflow` entirely, so an over-icon badge never gets clipped. Active = amber
// icon + a short amber marker along the top edge.
export const BOTTOM_TAB =
  'group/tab relative flex h-full flex-1 items-center justify-center rounded-none border-0 px-1 text-muted-foreground transition-colors data-[state=active]:bg-transparent data-[state=active]:text-[color:var(--grimorio-gold)] data-[state=active]:shadow-none data-[state=active]:before:absolute data-[state=active]:before:left-1/2 data-[state=active]:before:top-0 data-[state=active]:before:h-0.5 data-[state=active]:before:w-6 data-[state=active]:before:-translate-x-1/2 data-[state=active]:before:rounded-full data-[state=active]:before:bg-[color:var(--grimorio-gold)]'

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
  tab,
  onTabChange,
}: {
  character: Character
  barSlot?: ReactNode
  inSession?: boolean
  tab: string
  onTabChange: (value: string) => void
}) {
  const panels = SHEET_PANELS
  // From `md` the HUD's stat cluster (see CharacterHud) shows exactly the
  // Vitais content, so the tab would be a full duplicate — drop it (task 12).
  const hudShowsVitals = useMediaQuery('(min-width: 768px)')
  const sections = hudShowsVitals ? panels : [vitalsSection(), ...panels]
  const resolved = resolveSheetTab(tab)
  const active = sections.some((s) => s.value === resolved)
    ? resolved
    : sections[0]!.value

  return (
    <Tabs
      value={active}
      onValueChange={onTabChange}
      // In a session the sheet blends into the session bg; standalone keeps the
      // full-bleed sheet gradient. Panels carry their own surfaces either way.
      className={cn(
        'flex h-full min-h-0 w-full min-w-0 flex-col',
        !inSession && sheetBg,
      )}
    >
      {/* min-h-40 floor: on short viewports (phone landscape) the shrink-0
          HUD + tab bar would otherwise squeeze this flex-1 area to 0px and the
          active panel's first header (e.g. "Condições") vanished under the top
          nav (audit task 11). With a floor, the AppShell <main> scrolls
          instead, keeping the panel readable. */}
      <div className="min-h-40 min-w-0 flex-1 overflow-hidden">
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
      <CharacterHud character={character} className="shrink-0" />
      {/* Full-width bar; cells share the width (no scroll, no overflow) so the
          over-icon badge is never clipped. */}
      <div className="w-full shrink-0 border-t border-border/60 bg-card/95 backdrop-blur">
        <TabsList className="flex h-14 w-full items-stretch gap-0 rounded-none border-0 bg-transparent p-0 landscape:h-11">
          {sections.map((s) => (
            <TabsTrigger
              key={s.value}
              value={s.value}
              aria-label={s.label}
              title={s.label}
              className={cn(BOTTOM_TAB, s.dim?.(character) && 'opacity-40')}
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
