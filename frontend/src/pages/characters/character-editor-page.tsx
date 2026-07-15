import { getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Skeleton } from '@/shared/ui/skeleton'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'

const routeApi = getRouteApi('/characters/$id')

export function CharacterViewPage() {
  const { id } = routeApi.useParams()
  const character = useQuery(characterQueryOptions(Number(id)))

  if (character.isLoading) return <CharacterSheetSkeleton />
  if (character.isError) {
    return (
      <p className="p-6 text-destructive">{(character.error as Error).message}</p>
    )
  }
  if (!character.data) return null

  return <CharacterSheet character={character.data} />
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
