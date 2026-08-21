import { useQuery } from '@tanstack/solid-query'
import { ScrollText } from 'lucide-solid'
import { Show } from 'solid-js'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { NewSessionButton } from './new-session-button'
import { SessionLog } from './session-log'
import { TomeSection } from './tome-section'

/**
 * Sessões section: the campaign's chronicle log — every session as an entry on
 * a gilt timeline (the live one highlighted). The GM mints the next session
 * from the heading action; the empty state invites the first entry.
 */
export function SessionsCard(props: { campaignId: number; isGm: boolean }) {
  const sessions = useQuery(() => campaignSessionsQueryOptions(props.campaignId))
  const list = () => sessions.data ?? []

  return (
    <TomeSection
      eyebrow="Crônica"
      title="Sessões"
      action={<Show when={props.isGm}>{<NewSessionButton campaignId={props.campaignId} />}</Show>}
    >
      <Show when={sessions.isLoading}>
        <SkeletonRows count={3} />
      </Show>
      <Show when={!sessions.isLoading && list().length === 0}>
        <EmptyLog isGm={props.isGm} />
      </Show>
      <Show when={list().length > 0}>
        <SessionLog sessions={list()} campaignId={props.campaignId} />
      </Show>
    </TomeSection>
  )
}

/** No sessions yet — a blank page waiting for the first entry. */
function EmptyLog(props: { isGm: boolean }) {
  return (
    <div class="flex flex-col items-center gap-2 rounded-none border border-dashed border-grimorio-iron px-4 py-10 text-center">
      <ScrollText aria-hidden="true" class="size-6 text-muted-foreground" />
      <p class="text-sm text-muted-foreground">
        A crônica ainda não tem sessões.
        <Show when={props.isGm}> Abra a primeira para começar a registrar.</Show>
      </p>
    </div>
  )
}
