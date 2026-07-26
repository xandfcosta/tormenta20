import { type ReactNode, useEffect, useRef, useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { cn } from '@/shared/lib/utils'

/**
 * A focus request from the Pendências callout: which card to open, plus a
 * nonce so clicking the same pendência twice re-triggers the open + scroll.
 */
export type CardFocus = { id: string; nonce: number } | null

/**
 * Collapsible section shell for the Habilidades sub-tabs. Shows a title, an
 * optional count, and a "pendente" badge when the section still owes choices.
 * When `focus` targets this card's `id`, it opens and scrolls into view.
 */
export function CollapsibleAbilityCard({
  id,
  title,
  count,
  pending = 0,
  defaultOpen = true,
  focus,
  children,
}: {
  id: string
  title: string
  count?: ReactNode
  pending?: number
  defaultOpen?: boolean
  focus: CardFocus
  children: ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  const ref = useRef<HTMLDivElement>(null)
  // `focus` is a fresh object per jump (with a nonce), so depending on it
  // re-fires even when the same card is re-targeted after a collapse.
  useEffect(() => {
    if (focus?.id !== id) return
    setOpen(true)
    ref.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }, [focus, id])

  return (
    <div ref={ref} className="overflow-hidden rounded-lg border bg-card">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
        className="flex w-full items-center gap-2 rounded-t-lg px-3 py-2 text-left hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
      >
        <ChevronDown
          className={cn(
            'size-4 shrink-0 text-muted-foreground transition-transform',
            !open && '-rotate-90',
          )}
        />
        <span className="text-sm font-semibold">{title}</span>
        {count != null && (
          <span className="text-xs text-muted-foreground">{count}</span>
        )}
        {pending > 0 && (
          <Badge variant="destructive" className="ml-auto">
            {pending} pendente{pending > 1 ? 's' : ''}
          </Badge>
        )}
      </button>
      {open && <div className="border-t px-3 py-2">{children}</div>}
    </div>
  )
}
