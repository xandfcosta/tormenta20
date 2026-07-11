import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Badge } from '@/shared/ui/badge'
import { SkeletonRows } from '@/shared/ui/skeleton'
import { characterCampaignsQueryOptions } from '@/shared/lib/queries'

/**
 * Right-panel tab that lists every campaign a character is currently
 * a member of. Owner + description are surfaced so the player can
 * jump straight to the correct table from the sheet.
 */
export function CampaignsPanel({ characterId }: { characterId: number }) {
  const query = useQuery(characterCampaignsQueryOptions(characterId))
  if (query.isLoading) {
    return <SkeletonRows count={2} className="p-2" />
  }
  if (query.isError) {
    return (
      <p className="p-2 text-sm text-destructive">
        {(query.error as Error).message}
      </p>
    )
  }
  const rows = query.data ?? []
  if (rows.length === 0) {
    return (
      <p className="p-2 text-sm text-muted-foreground">
        Este personagem não participa de nenhuma campanha ainda.
      </p>
    )
  }
  return (
    <div className="flex flex-col gap-2 overflow-y-auto p-2">
      {rows.map((row) => (
        <Link
          key={row.id}
          to="/campaigns/$id"
          params={{ id: String(row.campaignId) }}
        >
          <div className="rounded-md border p-3 transition hover:border-primary/40">
            <div className="flex items-center justify-between">
              <p className="font-medium">{row.campaign.name}</p>
              <Badge variant={row.role === 'gm' ? 'default' : 'outline'}>
                {row.role === 'gm' ? 'GM' : 'Jogador'}
              </Badge>
            </div>
            {row.campaign.description && (
              <p className="mt-1 line-clamp-2 text-xs text-muted-foreground">
                {row.campaign.description}
              </p>
            )}
          </div>
        </Link>
      ))}
    </div>
  )
}
