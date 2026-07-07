import { CircleAlert, Loader, PlugZap, WifiOff } from 'lucide-react'
import { cn } from '@/lib/utils'

/**
 * ConnectionChip — persistent WebSocket state indicator. Sits in the
 * app shell so a player mid-combat can tell at a glance whether their
 * mutations are landing on the server. Responsive audit called out
 * that we lacked this — without it, silent WS drops just look like
 * a slow tracker.
 *
 * `dirty` piggybacks on the existing `persistence-warning` event —
 * shows a subtle warning even when the socket is technically live.
 */
type ConnectionStatus = 'connected' | 'reconnecting' | 'offline'

type ConnectionChipProps = {
  status: ConnectionStatus
  dirty?: boolean
  compact?: boolean
  className?: string
}

const STATUS_COPY: Record<ConnectionStatus, string> = {
  connected: 'Conectado',
  reconnecting: 'Reconectando…',
  offline: 'Offline',
}

function ConnectionChip({
  status,
  dirty = false,
  compact = false,
  className,
}: ConnectionChipProps) {
  const Icon =
    status === 'connected'
      ? dirty
        ? CircleAlert
        : PlugZap
      : status === 'reconnecting'
        ? Loader
        : WifiOff

  const tone =
    status === 'connected'
      ? dirty
        ? 'border-[color:var(--hp-hurt)]/60 bg-[color-mix(in_oklch,var(--hp-hurt)_10%,transparent)] text-[color:var(--hp-hurt)]'
        : 'border-[color:var(--hp-full)]/50 bg-[color-mix(in_oklch,var(--hp-full)_10%,transparent)] text-[color:var(--hp-full)]'
      : status === 'reconnecting'
        ? 'border-[color:var(--primary)]/60 bg-[color-mix(in_oklch,var(--primary)_10%,transparent)] text-[color:var(--primary)]'
        : 'border-[color:var(--hp-critical)]/70 bg-[color-mix(in_oklch,var(--hp-critical)_15%,transparent)] text-[color:var(--hp-critical)]'

  const label = dirty ? 'Alterações não salvas' : STATUS_COPY[status]

  return (
    <span
      role="status"
      data-slot="connection-chip"
      data-status={status}
      data-dirty={dirty ? 'true' : 'false'}
      aria-label={label}
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium',
        tone,
        className,
      )}
    >
      <Icon
        className={cn(
          'size-3.5',
          status === 'reconnecting' && 'animate-spin',
        )}
      />
      {!compact && <span>{label}</span>}
    </span>
  )
}

export { ConnectionChip }
export type { ConnectionStatus }
