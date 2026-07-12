import { getRouteApi } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { characterQueryOptions } from '@/entities/character/queries'
import { CharacterSheet } from '@/features/character-sheet/character-sheet'

const routeApi = getRouteApi('/characters/$id')

export function CharacterViewPage() {
  const { id } = routeApi.useParams()
  const character = useQuery(characterQueryOptions(Number(id)))

  if (character.isLoading) return <p className="p-6">Loading…</p>
  if (character.isError) {
    return (
      <p className="p-6 text-destructive">{(character.error as Error).message}</p>
    )
  }
  if (!character.data) return null

  return <CharacterSheet character={character.data} />
}
