import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { CalendarClock, Plus, UserPlus } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import type { Campaign } from '@/shared/api/api'

/**
 * Campaigns list, grouped by the caller's role: "Mestrando" (campaigns you
 * GM) and "Jogando" (joined as a player). Role is the primary axis, so the
 * section conveys it and the cards stay lean (name / description / updated).
 */
export function CampaignsListPage() {
  const campaigns = useQuery(campaignsQueryOptions)
  const mastering = campaigns.data?.filter((c) => c.role === 'gm')
  const playing = campaigns.data?.filter((c) => c.role !== 'gm')

  return (
    <PageChrome className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">Campanhas</h1>
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
        <p className="text-destructive">{(campaigns.error as Error).message}</p>
      )}
      {campaigns.data?.length === 0 && <NoCampaigns />}

      {mastering && mastering.length > 0 && (
        <CampaignSection title="Mestrando" campaigns={mastering} />
      )}
      {playing && playing.length > 0 && (
        <CampaignSection title="Jogando" campaigns={playing} />
      )}
    </PageChrome>
  )
}

/** A titled role section with a responsive card grid. */
function CampaignSection({
  title,
  campaigns,
}: {
  title: string
  campaigns: Campaign[]
}) {
  return (
    <section className="space-y-3">
      <h2 className="text-sm font-medium uppercase tracking-widest text-muted-foreground">
        {title}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {campaigns.map((c) => (
          <CampaignCard key={c.id} campaign={c} />
        ))}
      </div>
    </section>
  )
}

function CampaignCard({ campaign }: { campaign: Campaign }) {
  return (
    <Link to="/campaigns/$id" params={{ id: String(campaign.id) }}>
      <Card className="h-full transition-colors hover:border-primary">
        <CardHeader>
          <CardTitle>{campaign.name}</CardTitle>
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

function NoCampaigns() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
        <p>Nenhuma campanha ainda.</p>
        <Link to="/campaigns/new">
          <Button>Criar sua primeira campanha</Button>
        </Link>
      </CardContent>
    </Card>
  )
}
