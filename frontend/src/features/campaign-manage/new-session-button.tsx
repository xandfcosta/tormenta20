import { useQuery, useQueryClient } from '@tanstack/solid-query'
import { useNavigate } from '@tanstack/solid-router'
import { CalendarPlus } from 'lucide-solid'
import { createSignal } from 'solid-js'
import { campaignQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { type Session, api } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'

/**
 * The number the next session gets: one past the HIGHEST used, not the count —
 * deleting session 2 of [1,2,3] would make the count collide with an existing
 * number.
 *
 * @example nextSessionNumber([{ sessionNumber: 3 }, { sessionNumber: 1 }]) // 4
 */
export function nextSessionNumber(sessions: readonly Session[]): number {
  return sessions.reduce((max, s) => Math.max(max, s.sessionNumber), 0) + 1
}

export type NewSessionActionProps = {
  nextNumber: number
  /** Overrides the default "Sessão N" — used by the empty state's invitation. */
  label?: string
  onCreate: () => Promise<void>
  size?: 'sm' | 'default' | 'lg'
  variant?: 'default' | 'outline'
  class?: string
}

/**
 * The button that mints the next session. Presentational: it owns the in-flight
 * lock (a second click would open a duplicate) and nothing else.
 *
 * @example <NewSessionAction nextNumber={4} onCreate={create} />
 */
export function NewSessionAction(props: NewSessionActionProps) {
  const [pending, setPending] = createSignal(false)

  const create = async () => {
    setPending(true)
    try {
      await props.onCreate()
    } finally {
      setPending(false)
    }
  }

  return (
    <Button
      size={props.size ?? 'sm'}
      variant={props.variant ?? 'default'}
      class={props.class}
      disabled={pending()}
      onClick={create}
    >
      <CalendarPlus aria-hidden="true" class="mr-1 size-4" />
      {pending() ? 'Criando…' : (props.label ?? `Sessão ${props.nextNumber}`)}
    </Button>
  )
}

export type NewSessionButtonProps = Omit<NewSessionActionProps, 'nextNumber' | 'onCreate'> & {
  campaignId: number
}

/**
 * Creates the next session and walks straight into it. Shared by the Sessões
 * heading and the Visão geral empty state so the create flow lives in one place.
 */
export function NewSessionButton(props: NewSessionButtonProps) {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const sessions = useQuery(() => campaignSessionsQueryOptions(props.campaignId))

  const create = async () => {
    const created = await api.sessions.create(props.campaignId, {
      sessionNumber: nextSessionNumber(sessions.data ?? []),
    })
    // The campaign key is the parent of `sessions`, so one invalidation sweeps
    // the list AND the overview's counters.
    await queryClient.invalidateQueries({
      queryKey: campaignQueryOptions(props.campaignId).queryKey,
    })
    await navigate({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: String(props.campaignId), sid: String(created.id) },
    })
  }

  return (
    <NewSessionAction
      nextNumber={nextSessionNumber(sessions.data ?? [])}
      label={props.label}
      size={props.size}
      variant={props.variant}
      class={props.class}
      onCreate={create}
    />
  )
}
