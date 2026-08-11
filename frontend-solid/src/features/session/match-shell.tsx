import { Link } from '@tanstack/solid-router'
import { LogOut } from 'lucide-solid'
import type { JSX } from 'solid-js'
import { buttonVariants } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'

/**
 * Full-screen frame for a live session ("match mode"): a slim bar with the
 * session identity, presence and the way out, over a scrollable body. The
 * role-specific views render as children.
 */
export function MatchShell(props: {
  campaignId: number
  title: string
  /** Right-hand slot — presence chips today. */
  bar?: JSX.Element
  children: JSX.Element
}) {
  return (
    // `scene-grimorio` is the TOKEN SCOPE, not a skin: without it the whole
    // match renders in shadcn's light defaults. Match mode owns the viewport,
    // so it carries the scope itself instead of going through SceneShell,
    // which brings a back button and a title bar this screen already has.
    <div class="scene-grimorio flex h-dvh min-h-0 flex-col bg-background text-foreground">
      <header class="flex items-center justify-between gap-3 border-b border-grimorio-iron bg-[var(--grimorio-panel)] px-3 py-2 sm:px-4">
        <p class="min-w-0 flex-1 truncate font-heading tracking-wide text-grimorio-gold">
          {props.title}
        </p>
        <div class="flex items-center gap-2">
          {props.bar}
          {/* No `asChild` in Solid: a link that looks like a button IS a link
              wearing the button classes. */}
          <Link
            to="/campaigns/$id"
            params={{ id: String(props.campaignId) }}
            class={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'gap-1.5')}
          >
            <LogOut aria-hidden="true" class="size-4" />
            <span class="hidden sm:inline">Sair da sessão</span>
          </Link>
        </div>
      </header>
      <div class="min-h-0 flex-1 overflow-y-auto">{props.children}</div>
    </div>
  )
}
