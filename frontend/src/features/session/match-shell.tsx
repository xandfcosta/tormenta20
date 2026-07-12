import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import { LogOut } from 'lucide-react'
import { Button } from '@/shared/ui/button'

/**
 * Full-screen frame for a live session ("match mode"). Renders inside the
 * bare AppShell (no app nav), so it owns the whole viewport: a slim
 * session bar (identity + presence + leave) over a scrollable body. The
 * role-specific views (GM tracker / player sheet) render as `children`.
 */
export function MatchShell({
  campaignId,
  title,
  bar,
  children,
}: {
  campaignId: number
  title: string
  /** Right-hand bar slot — presence chips today, more later. */
  bar?: ReactNode
  children: ReactNode
}) {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex items-center justify-between gap-3 border-b border-border/60 bg-card/70 px-3 py-2 backdrop-blur sm:px-4">
        <p className="min-w-0 flex-1 truncate font-display tracking-wide">
          {title}
        </p>
        <div className="flex items-center gap-2">
          {bar}
          <Link to="/campaigns/$id" params={{ id: String(campaignId) }}>
            <Button variant="outline" size="sm" className="gap-1.5">
              <LogOut className="size-4" />
              <span className="hidden sm:inline">Sair da sessão</span>
            </Button>
          </Link>
        </div>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
    </div>
  )
}
