import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader } from '@/shared/ui/card'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { campaignMembersQueryOptions } from '@/entities/campaign/queries'
import { campaignSessionsQueryOptions } from '@/entities/session/queries'
import { InviteButton } from './invite-button'
import { NewSessionButton } from './new-session-button'

/**
 * Campaign "Visão geral" tab (ALE-29): the current-session block up top (the
 * one thing a GM opens the campaign to do) plus light summaries of Membros and
 * Sessões that link into their own tabs. Replaces the old duplicated
 * active-session banner.
 */
export function CampaignOverview({
  campaignId,
  isGm,
  onGoToTab,
}: {
  campaignId: number
  isGm: boolean
  onGoToTab: (tab: 'sessoes' | 'membros') => void
}) {
  return (
    <div className="space-y-6">
      <CurrentSessionCard campaignId={campaignId} isGm={isGm} />
      <div className="grid gap-6 sm:grid-cols-2">
        <MembersSummary
          campaignId={campaignId}
          isGm={isGm}
          onSeeAll={() => onGoToTab('membros')}
        />
        <SessionsSummary
          campaignId={campaignId}
          onSeeAll={() => onGoToTab('sessoes')}
        />
      </div>
    </div>
  )
}

function CurrentSessionCard({
  campaignId,
  isGm,
}: {
  campaignId: number
  isGm: boolean
}) {
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const active = sessions.data?.find((s) => s.status === 'active')

  if (active) {
    return (
      <Card className="border-primary bg-primary/5">
        <CardContent className="flex flex-wrap items-center justify-between gap-3 pt-6">
          <div>
            <p className="flex items-center gap-2 font-medium">
              <span className="size-2 shrink-0 rounded-full bg-[color:var(--hp-full)]" />
              Sessão {active.sessionNumber} em andamento
            </p>
            {active.startedAt && (
              <p className="text-xs text-muted-foreground">
                Iniciada{' '}
                {new Date(active.startedAt).toLocaleString('pt-BR', {
                  dateStyle: 'short',
                  timeStyle: 'short',
                })}
              </p>
            )}
          </div>
          <Link
            to="/campaigns/$id/sessions/$sid"
            params={{ id: campaignId, sid: active.id }}
          >
            <Button>Entrar →</Button>
          </Link>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center gap-3 py-8 text-center">
        <p className="text-sm text-muted-foreground">
          Nenhuma sessão ao vivo no momento.
        </p>
        {isGm && (
          <NewSessionButton
            campaignId={campaignId}
            label="Nova sessão"
            size="default"
          />
        )}
      </CardContent>
    </Card>
  )
}

function MembersSummary({
  campaignId,
  isGm,
  onSeeAll,
}: {
  campaignId: number
  isGm: boolean
  onSeeAll: () => void
}) {
  const members = useQuery(campaignMembersQueryOptions(campaignId))
  const count = members.data?.length ?? 0

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between">
        <SectionHeading as="h2">Membros</SectionHeading>
        {isGm && <InviteButton campaignId={campaignId} />}
      </CardHeader>
      <CardContent className="space-y-2">
        {members.isLoading && <SkeletonRows count={2} />}
        {count === 0 && (
          <p className="text-sm text-muted-foreground">
            Nenhum personagem inscrito ainda.
          </p>
        )}
        {members.data?.slice(0, 3).map((m) => (
          <div key={m.id} className="flex items-center gap-2 text-sm">
            <span className="font-medium">
              {m.character?.name ?? `Personagem ${m.characterId}`}
            </span>
            <Badge variant={m.role === 'gm' ? 'default' : 'outline'}>
              {m.role === 'gm' ? 'GM' : 'Jogador'}
            </Badge>
          </div>
        ))}
        {count > 0 && (
          <Button variant="ghost" size="sm" onClick={onSeeAll}>
            Ver {count} {count === 1 ? 'membro' : 'membros'} →
          </Button>
        )}
      </CardContent>
    </Card>
  )
}

function SessionsSummary({
  campaignId,
  onSeeAll,
}: {
  campaignId: number
  onSeeAll: () => void
}) {
  const sessions = useQuery(campaignSessionsQueryOptions(campaignId))
  const recent = [...(sessions.data ?? [])]
    .sort((a, b) => b.sessionNumber - a.sessionNumber)
    .slice(0, 3)

  return (
    <Card>
      <CardHeader>
        <SectionHeading as="h2">Sessões recentes</SectionHeading>
      </CardHeader>
      <CardContent className="space-y-2">
        {sessions.isLoading && <SkeletonRows count={2} />}
        {recent.length === 0 && (
          <p className="text-sm text-muted-foreground">Nenhuma sessão ainda.</p>
        )}
        {recent.map((s) => (
          <div
            key={s.id}
            className="flex items-center justify-between text-sm"
          >
            <span>
              Sessão {s.sessionNumber}
              {s.title && (
                <span className="text-muted-foreground"> — {s.title}</span>
              )}
            </span>
            <Badge
              variant={
                s.status === 'active'
                  ? 'default'
                  : s.status === 'ended'
                    ? 'secondary'
                    : 'outline'
              }
            >
              {s.status === 'planned'
                ? 'Planejada'
                : s.status === 'active'
                  ? 'Ativa'
                  : 'Encerrada'}
            </Badge>
          </div>
        ))}
        <Button variant="ghost" size="sm" onClick={onSeeAll}>
          Ver todas →
        </Button>
      </CardContent>
    </Card>
  )
}
