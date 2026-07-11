import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { CalendarClock, Plus, UserPlus } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { meQueryOptions } from '@/entities/user/queries'
import type { Campaign } from '@/shared/api/api'

export const Route = createFileRoute('/campaigns/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(campaignsQueryOptions),
  component: CampaignsListPage,
})

function CampaignsListPage() {
  const campaigns = useQuery(campaignsQueryOptions)

  return (
    <PageChrome className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SectionHeading variant="aharadak" as="h1">
          Campanhas
        </SectionHeading>
        <div className="flex gap-2">
          <Link to="/campaigns/join">
            <Button variant="outline">
              <UserPlus className="mr-1 size-4" /> Entrar em campanha
            </Button>
          </Link>
          <Link to="/campaigns/new">
            <Button>
              <Plus className="mr-1 size-4" /> Nova campanha
            </Button>
          </Link>
        </div>
      </div>

      {campaigns.isLoading && <SkeletonCardGrid count={3} />}
      {campaigns.isError && (
        <p className="text-destructive">
          {(campaigns.error as Error).message}
        </p>
      )}
      {campaigns.data?.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
            <p>Nenhuma campanha ainda.</p>
            <Link to="/campaigns/new">
              <Button>Criar sua primeira campanha</Button>
            </Link>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {campaigns.data?.map((c) => (
          <CampaignCard key={c.id} campaign={c} />
        ))}
      </div>
    </PageChrome>
  )
}

function CampaignCard({ campaign }: { campaign: Campaign }) {
  return (
    <Link to="/campaigns/$id" params={{ id: String(campaign.id) }}>
      <Card className="h-full transition hover:border-[color:var(--primary)]/50 hover:shadow-[0_1px_0_rgba(0,0,0,0.04),0_18px_50px_-30px_var(--primary)]">
        <CardHeader>
          <CardTitle className="font-display text-lg tracking-wide">
            {campaign.name}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          {campaign.description && (
            <p className="line-clamp-3 text-muted-foreground">
              {campaign.description}
            </p>
          )}
          <p className="flex items-center gap-1 text-xs text-muted-foreground">
            <CalendarClock className="size-3" />
            Atualizada em{' '}
            {new Date(campaign.updatedAt).toLocaleDateString('pt-BR')}
          </p>
        </CardContent>
      </Card>
    </Link>
  )
}
