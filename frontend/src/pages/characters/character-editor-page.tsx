import { getRouteApi, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'
import { SceneShell } from '@/shared/layout/scene-shell'
import { useSfx } from '@/shared/lib/use-sfx'
import { Skeleton } from '@/shared/ui/skeleton'

const routeApi = getRouteApi('/characters/$id')

/**
 * Standalone character sheet, rendered as a full grimório scene. The sheet
 * itself (`CharacterSheet`) is shared with the in-session player view, which
 * renders it WITHOUT this SceneShell — so only the standalone page picks up the
 * `.scene-grimorio` scope; in-session stays on plain shadcn until the Mesa
 * epic. `bleed` hands the full height to the sheet's HUD-pinned grid.
 */
export function CharacterViewPage() {
  const { id } = routeApi.useParams()
  const navigate = useNavigate()
  const sfx = useSfx()
  const character = useQuery(characterQueryOptions(Number(id)))
  const data = character.data

  return (
    <SceneShell
      bleed
      dense
      title={data?.name ?? 'Ficha'}
      onBack={() => {
        sfx('select')
        navigate({ to: '/characters' })
      }}
      onEnter={() => sfx('transition')}
    >
      {character.isLoading ? (
        <CharacterSheetSkeleton />
      ) : character.isError ? (
        <p className="p-6 text-destructive">
          {(character.error as Error).message}
        </p>
      ) : data ? (
        <CharacterSheet character={data} />
      ) : null}
    </SceneShell>
  )
}

/** Matches the sheet's header-bar + aside/main split so nothing jumps. */
function CharacterSheetSkeleton() {
  return (
    <div className="space-y-4 p-6">
      <Skeleton className="h-9 w-64" />
      <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
        <Skeleton className="h-64 w-full" />
        <div className="space-y-3">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      </div>
    </div>
  )
}
