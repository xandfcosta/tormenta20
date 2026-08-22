import { Users } from 'lucide-solid'
import { For, Show } from 'solid-js'
import type { PresenceUser } from '@/shared/realtime/realtime'
import { cn } from '@/shared/lib/utils'

/**
 * "Who's online" chips for the live session, from the realtime `presence`
 * roster (deduped by user server-side). The GM's chip is filled so the mestre
 * stands out from the players at a glance.
 */
export function PresenceChips(props: { users: PresenceUser[] }) {
  return (
    <Show when={props.users.length > 0}>
      <div class="flex flex-wrap items-center gap-1 text-xs">
        <Users aria-label="Conectados" class="size-3.5 text-muted-foreground" />
        <For each={props.users}>
          {(user) => (
            <span
              class={cn(
                'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-2xs',
                user.role === 'gm'
                  ? 'bg-marker text-marker-foreground'
                  : 'bg-muted text-foreground',
              )}
            >
              <span class="size-1.5 rounded-full bg-[color:var(--hp-full)]" />
              {user.name}
            </span>
          )}
        </For>
      </div>
    </Show>
  )
}
