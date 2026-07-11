import { createFileRoute, Link, redirect } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { HpBar } from '@/shared/ui/hp-bar'
import { MpBar } from '@/shared/ui/mp-bar'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SectionHeading } from '@/shared/ui/section-heading'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { charactersQueryOptions } from '@/entities/character/queries'
import { meQueryOptions } from '@/entities/user/queries'
import type { Character } from '@/shared/api/api'

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
    <PageChrome className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <SectionHeading variant="kallyadranoch" as="h1">
          Personagens
        </SectionHeading>
        <Link to="/characters/new">
          <Button>
            <Plus className="mr-1 size-4" /> Novo personagem
          </Button>
        </Link>
      </div>

      {characters.isLoading && <SkeletonCardGrid count={3} />}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {characters.data?.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
            <p>Nenhum personagem ainda.</p>
            <Link to="/characters/new">
              <Button>Criar seu primeiro personagem</Button>
            </Link>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {characters.data?.map((c) => <CharacterCard key={c.id} character={c} />)}
      </div>
    </PageChrome>
  )
}

function CharacterCard({ character }: { character: Character }) {
  const classes = character.classes.map((c) => `${c.className} ${c.level}`).join(' / ')
  const races = character.races.map((r) => r.race).join(', ')

  return (
    <Link to="/characters/$id" params={{ id: String(character.id) }}>
      <Card className="h-full transition hover:border-[color:var(--primary)]/50 hover:shadow-[0_1px_0_rgba(0,0,0,0.04),0_18px_50px_-30px_var(--primary)]">
        <CardHeader>
          <CardTitle className="flex items-center justify-between font-display tracking-wide">
            <span>{character.name}</span>
            <Badge variant="secondary">Nv {character.level}</Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <p className="text-muted-foreground">{races} • {character.origin}</p>
          <p className="font-medium">{classes}</p>
          <div className="space-y-1.5 pt-1">
            <HpBar
              current={character.hpCurrent}
              max={character.hpMax}
              size="sm"
              label="PV"
            />
            <MpBar
              current={character.mpCurrent}
              max={character.mpMax}
              size="sm"
              label="PM"
            />
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
