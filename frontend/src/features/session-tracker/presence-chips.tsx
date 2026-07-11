import { Users } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import type { PresenceUser } from '@/shared/realtime/realtime'

/**
 * "Who's online" chips for the live session. Fed by the realtime
 * `presence` roster (deduped by user server-side). GM chips use the
 * primary variant so the mestre stands out from players at a glance.
 */
export function PresenceChips({ users }: { users: PresenceUser[] }) {
  if (users.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-1 text-xs">
      <Users
        className="size-3.5 text-muted-foreground"
        aria-label="Conectados"
      />
      {users.map((u) => (
        <Badge
          key={u.userId}
          variant={u.role === 'gm' ? 'default' : 'secondary'}
          className="gap-1"
        >
          <span className="size-1.5 rounded-full bg-[color:var(--hp-full)]" />
          {u.name}
        </Badge>
      ))}
    </div>
  )
}
