import { type ReactNode, useState } from 'react'
import { ChevronUp, Swords } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/shared/ui/sheet'
import { useMediaQuery } from '@/shared/lib/use-media-query'
import type { useSessionSocket } from '@/shared/realtime/realtime'

/**
 * Session side rail that adapts to viewport. On desktop (`lg+`) it's an
 * aside column beside the main surface. On phones the main surface (tracker
 * or sheet) owns the screen and the rail collapses into a bottom sheet: a
 * fixed bar shows a live `peek` (current turn / round) and drags up into the
 * full controls. Rendered once — a media query, not duplicated DOM — so the
 * interactive cards inside keep a single mount.
 */
export function MatchRail({
  title,
  peek,
  children,
}: {
  title: string
  /** Compact live status shown in the collapsed phone bar. */
  peek?: ReactNode
  children: ReactNode
}) {
  const isDesktop = useMediaQuery('(min-width: 1024px)')

  if (isDesktop)
    return (
      <aside className="max-h-full space-y-4 overflow-y-auto">{children}</aside>
    )

  return (
    <div className="fixed inset-x-0 bottom-0 z-40 flex items-center gap-3 border-t border-border/60 bg-card/90 px-3 py-2 backdrop-blur">
      <div className="min-w-0 flex-1 truncate text-sm text-muted-foreground">
        {peek}
      </div>
      <MatchControls title={title}>{children}</MatchControls>
    </div>
  )
}

/**
 * The rail's collapsible controls, decoupled from the fixed bar: a trigger
 * that opens the session controls in a bottom sheet. Used inside MatchRail's
 * phone bar and, in the player view, merged into the character sheet's own
 * mobile bottom bar so a phone shows a single bar, not two stacked.
 */
export function MatchControls({
  title,
  trigger,
  children,
}: {
  title: string
  /** Custom trigger; defaults to an outline "chevron + title" button. */
  trigger?: ReactNode
  children: ReactNode
}) {
  const [open, setOpen] = useState(false)
  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        {trigger ?? (
          <Button size="sm" variant="outline" className="gap-1.5">
            <ChevronUp className="size-4" />
            {title}
          </Button>
        )}
      </SheetTrigger>
      <SheetContent
        side="bottom"
        className="flex max-h-[85vh] flex-col gap-0 rounded-t-xl"
      >
        <SheetHeader>
          <SheetTitle className="font-display tracking-wide">{title}</SheetTitle>
        </SheetHeader>
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          {children}
        </div>
      </SheetContent>
    </Sheet>
  )
}

/** Live one-liner for the collapsed phone bar: round + who's up. */
export function MatchPeek({
  rt,
}: {
  rt: ReturnType<typeof useSessionSocket>
}) {
  const active =
    rt.state.turnIndex >= 0 ? rt.state.initiative[rt.state.turnIndex] : undefined
  return (
    <span className="flex items-center gap-1.5">
      <span className="font-hud tabular-nums">Rodada {rt.state.round}</span>
      {active && (
        <>
          <Swords className="size-3.5 text-[color:var(--primary)]" />
          <span className="truncate">{active.label}</span>
        </>
      )}
    </span>
  )
}
