import { CircleAlert, Loader, PlugZap, WifiOff } from 'lucide-solid'
import { Dynamic } from 'solid-js/web'
import { cn } from '@/shared/lib/utils'

export type ConnectionStatus = 'connected' | 'reconnecting' | 'offline'

const STATUS_COPY: Record<ConnectionStatus, string> = {
  connected: 'Conectado',
  reconnecting: 'Reconectando…',
  offline: 'Offline',
}

/**
 * Live WebSocket state, in words. Without it a silent drop mid-combat just
 * looks like a slow tracker — the player has no way to tell that their hits
 * are landing nowhere.
 *
 * `dirty` rides the server's `persistence-warning`: the socket is up, but the
 * last mutation did not persist.
 *
 * @example <ConnectionChip status="reconnecting" dirty={false} />
 */
export function ConnectionChip(props: {
  status: ConnectionStatus
  dirty?: boolean
  class?: string
}) {
  const icon = () => {
    if (props.status === 'connected') return props.dirty ? CircleAlert : PlugZap
    return props.status === 'reconnecting' ? Loader : WifiOff
  }
  const tone = () => {
    if (props.status === 'connected') {
      return props.dirty
        ? 'border-amber-500/40 text-amber-300'
        : 'border-emerald-500/40 text-emerald-300'
    }
    return props.status === 'reconnecting'
      ? 'border-amber-500/40 text-amber-300'
      : 'border-destructive/50 text-red-300'
  }
  const label = () =>
    props.dirty && props.status === 'connected'
      ? 'Conectado · alterações não salvas'
      : STATUS_COPY[props.status]

  return (
    <span
      class={cn(
        'inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-widest',
        tone(),
        props.class,
      )}
      // role=status: the state CHANGES while the player is looking elsewhere,
      // and a drop is exactly what has to be announced.
      role="status"
      // `status` does NOT take its name from its contents (that rule covers
      // button/link/heading, not live regions), so without this the region is
      // announced nameless. The port dropped it and only the E2E suite noticed.
      aria-label={label()}
    >
      <Dynamic
        component={icon()}
        aria-hidden="true"
        class={cn('size-3', props.status === 'reconnecting' && 'animate-spin')}
      />
      {label()}
    </span>
  )
}
