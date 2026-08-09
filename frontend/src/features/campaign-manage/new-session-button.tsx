import { useNavigate } from '@tanstack/react-router'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarPlus } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { api } from '@/shared/api/api'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'

/**
 * Creates the next session (auto-numbered) and navigates straight into it.
 * Shared by the Sessões list header and the Visão geral empty state so the
 * create flow lives in one place (ALE-29). `label` omitted → "Sessão N".
 */
export function NewSessionButton({
  campaignId,
  label,
  size = 'sm',
  variant = 'default',
  className,
}: {
  campaignId: number
  label?: string
  size?: 'sm' | 'default' | 'lg'
  variant?: 'default' | 'outline'
  className?: string
}) {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const nextNumber =
    (sessions.data?.reduce((max, s) => Math.max(max, s.sessionNumber), 0) ?? 0) +
    1

  const mutation = useMutation({
    mutationFn: () =>
      api.sessions.create(campaignId, { sessionNumber: nextNumber }),
    onSuccess: async (created) => {
      qc.invalidateQueries({
        queryKey: campaignSessionsQueryOptions(campaignId).queryKey,
      })
      await navigate({
        to: '/campaigns/$id/sessions/$sid',
        params: { id: campaignId, sid: created.id },
      })
    },
  })

  return (
    <Button
      size={size}
      variant={variant}
      className={className}
      disabled={mutation.isPending}
      onClick={() => mutation.mutate()}
    >
      <CalendarPlus className="mr-1 size-4" />
      {mutation.isPending ? 'Criando…' : (label ?? `Sessão ${nextNumber}`)}
    </Button>
  )
}
