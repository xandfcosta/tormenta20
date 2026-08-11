import { getRouteApi } from '@tanstack/react-router'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/shared/ui/button'
import { Badge } from '@/shared/ui/badge'
import { PageChrome } from '@/shared/ui/page-chrome'
import { Skeleton } from '@/shared/ui/skeleton'
import { characterQueryOptions } from '@/entities/character/queries'
import { computedSheetV1For } from '@/entities/character/sheet-v1'
import { ComputedSheetCards } from '@/features/character-sheet/computed-sheet'

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
  const query = useQuery(characterQueryOptions(Number(id)))

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

  // Derived here (WASM, same Go engine) — the /sheet endpoint returns a
  // different payload than these cards read (ALE-77).
  const computed = computedSheetV1For(query.data)

  return (
    <PageChrome className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <Link to="/characters/$id" params={{ id }}>
          <Button variant="outline" size="sm">
            ← Voltar
          </Button>
        </Link>
        <h1 className="flex items-center gap-2 text-2xl font-bold tracking-tight">
          {query.data.name}
          <Badge variant="secondary">Nv {computed.level}</Badge>
        </h1>
      </div>

      <ComputedSheetCards computed={computed} showWarnings />
    </PageChrome>
  )
}
