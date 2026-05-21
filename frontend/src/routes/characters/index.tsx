import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { meQueryOptions, charactersQueryOptions } from '@/lib/queries'
import type { Character } from '@/lib/api'

export const Route = createFileRoute('/characters/')({
  beforeLoad: async ({ context, location }) => {
    const user = await context.queryClient.ensureQueryData(meQueryOptions)
    if (!user) throw redirect({ to: '/login', search: { redirect: location.href } })
  },
  loader: ({ context }) => context.queryClient.ensureQueryData(charactersQueryOptions),
  component: CharactersListPage,
})

function CharactersListPage() {
  const characters = useQuery(charactersQueryOptions)

  return (
    <div className="h-full space-y-6 overflow-y-auto p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-semibold">Characters</h1>
        <Link to="/characters/new">
          <Button>+ New character</Button>
        </Link>
      </div>

      {characters.isLoading && <p>Loading…</p>}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {characters.data?.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
            <p>No characters yet.</p>
            <Link to="/characters/new">
              <Button>Create your first character</Button>
            </Link>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {characters.data?.map((c) => <CharacterCard key={c.id} character={c} />)}
      </div>
    </div>
  )
}

function CharacterCard({ character }: { character: Character }) {
  const classes = character.classes.map((c) => `${c.className} ${c.level}`).join(' / ')
  const races = character.races.map((r) => r.race).join(', ')

  return (
    <Link to="/characters/$id" params={{ id: String(character.id) }}>
      <Card className="transition hover:border-primary/40 hover:shadow-md">
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>{character.name}</span>
            <Badge variant="secondary">Lv {character.level}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p className="text-muted-foreground">{races} • {character.origin}</p>
          <p className="font-medium">{classes}</p>
          <div className="flex gap-3 pt-2 text-xs">
            <span className="text-red-500">HP {character.hpCurrent}/{character.hpMax}</span>
            <span className="text-blue-500">MP {character.mpCurrent}/{character.mpMax}</span>
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
