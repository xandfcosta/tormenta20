import { getRouteApi } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { Skeleton } from '@/shared/ui/skeleton'
import { characterSheetQueryOptions } from '@/entities/character/queries'
import { ComputedSheetCards } from '@/features/character-sheet/computed-sheet'
import type { CharacterWithComputed } from '@/shared/api/api'

const routeApi = getRouteApi('/characters/$id/sheet')

/**
 * Server-computed sheet view. Renders the ComputedSheet payload from
 * `GET /characters/:id/sheet` — the same fields the orchestrator now
 * produces (attrs+race+vitals+Defesa full+saves+skills+attacks+movement).
 *
 * Separate from the main editor page: this one is read-only + a
 * consistency check that the mapper + orchestrator are talking correctly.
 * The card stack itself lives in `ComputedSheetCards` so the in-session
 * player view can render the identical sheet.
 */

export function CharacterSheetPage() {
  const { id } = routeApi.useParams()
  const query = useQuery(characterSheetQueryOptions(Number(id)))

  if (query.isLoading)
    return (
      <PageChrome className="space-y-3">
        <Skeleton className="h-8 w-72" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
      </PageChrome>
    )
  if (query.isError)
    return (
      <PageChrome>
        <p className="text-destructive">{(query.error as Error).message}</p>
      </PageChrome>
    )
  if (!query.data) return null

  const { computed } = query.data as CharacterWithComputed

  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <Link to="/characters/$id" params={{ id }}>
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
        <SectionHeading variant="kallyadranoch" as="h1" className="text-2xl">
          <span>
            {query.data.name}{' '}
            <Badge variant="secondary">Nv {computed.level}</Badge>
          </span>
        </SectionHeading>
      </div>

      <ComputedSheetCards computed={computed} showWarnings />
    </PageChrome>
  )
}
