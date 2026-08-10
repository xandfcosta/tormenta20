import { useQuery } from '@tanstack/react-query'
import { ScrollText } from 'lucide-react'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { NewSessionButton } from './new-session-button'
import { SessionLog } from './session-log'
import { TomeSection } from './tome-section'

/**
 * Sessões section: the campaign's chronicle log — every session as an entry on a
 * gilt timeline (the live one first + highlighted). GM mints the next session
 * from the heading action; empty state invites the first entry.
 */
export function SessionsCard({
  campaignId,
  isGm,
}: {
  campaignId: number
  isGm: boolean
}) {
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const list = sessions.data ?? []

  return (
    <TomeSection
      eyebrow="Crônica"
      title="Sessões"
      action={isGm && <NewSessionButton campaignId={campaignId} />}
    >
      {sessions.isLoading && <SkeletonRows count={3} />}
      {!sessions.isLoading && list.length === 0 && <EmptyLog isGm={isGm} />}
      {list.length > 0 && <SessionLog sessions={list} campaignId={campaignId} />}
    </TomeSection>
  )
}

/** No sessions yet — a blank page waiting for the first entry. */
function EmptyLog({ isGm }: { isGm: boolean }) {
  return (
    <div className="flex flex-col items-center gap-2 rounded-sm border border-dashed border-grimorio-iron px-4 py-10 text-center">
      <ScrollText aria-hidden className="size-6 text-muted-foreground" />
      <p className="text-sm text-muted-foreground">
        A crônica ainda não tem sessões.
        {isGm && ' Abra a primeira para começar a registrar.'}
      </p>
    </div>
  )
}
