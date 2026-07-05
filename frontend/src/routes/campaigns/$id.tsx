import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { campaignQueryOptions, meQueryOptions } from '@/lib/queries'

export const Route = createFileRoute('/campaigns/$id')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user)
      throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context, params }) =>
    context.queryClient.ensureQueryData(
      campaignQueryOptions(Number(params.id)),
    ),
  component: CampaignDetailPage,
})

/**
 * Placeholder detail page. Full edit/delete + sessions + members
 * management arrive in Fase D3. This ships alongside D2 so the "create
 * → navigate to detail" flow lands on a real screen instead of a 404.
 */
function CampaignDetailPage() {
  const { id } = Route.useParams()
  const campaign = useQuery(campaignQueryOptions(Number(id)))

  return (
    <div className="h-full space-y-6 overflow-y-auto p-6">
      <div className="flex items-center justify-between">
        <Link to="/campaigns">
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
      </div>

      {campaign.isLoading && <p>Loading…</p>}
      {campaign.isError && (
        <p className="text-destructive">
          {(campaign.error as Error).message}
        </p>
      )}
      {campaign.data && (
        <Card>
          <CardHeader>
            <CardTitle>{campaign.data.name}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            {campaign.data.description && (
              <p className="whitespace-pre-line text-muted-foreground">
                {campaign.data.description}
              </p>
            )}
            <p className="text-xs text-muted-foreground">
              Criada em{' '}
              {new Date(campaign.data.createdAt).toLocaleDateString('pt-BR')}
            </p>
            <p className="text-xs text-muted-foreground">
              Edição, sessões e membros virão na próxima fase.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
