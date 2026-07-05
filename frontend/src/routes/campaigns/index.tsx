import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { SkeletonCardGrid } from '@/components/ui/skeleton'
import { campaignsQueryOptions, meQueryOptions } from '@/lib/queries'
import type { Campaign } from '@/lib/api'

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
    <div className="h-full space-y-6 overflow-y-auto p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold">Campanhas</h1>
        <Link to="/campaigns/new">
          <Button>+ Nova campanha</Button>
        </Link>
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
    </div>
  )
}

function CampaignCard({ campaign }: { campaign: Campaign }) {
  return (
    <Link to="/campaigns/$id" params={{ id: String(campaign.id) }}>
      <Card className="transition hover:border-primary/40 hover:shadow-md">
        <CardHeader>
          <CardTitle>{campaign.name}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          {campaign.description && (
            <p className="line-clamp-3 text-muted-foreground">
              {campaign.description}
            </p>
          )}
          <p className="text-xs text-muted-foreground">
            Atualizada em{' '}
            {new Date(campaign.updatedAt).toLocaleDateString('pt-BR')}
          </p>
        </CardContent>
      </Card>
    </Link>
  )
}
