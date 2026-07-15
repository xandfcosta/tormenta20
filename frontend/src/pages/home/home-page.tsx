import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import type { LucideIcon } from 'lucide-react'
import { Plus, Scroll, Users2, Wand2 } from 'lucide-react'
import { Badge } from '@/shared/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SkeletonRows } from '@/shared/ui/skeleton'
import type { Campaign, Character } from '@/shared/api/api'
import { charactersQueryOptions } from '@/entities/character/queries'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import { meQueryOptions } from '@/entities/user/queries'

/**
 * Authed dashboard — the app's home base (`/`). A greeting, three shortcut
 * tiles into the main areas, then two overview columns (your characters /
 * your campaigns) for one-click resume. Anonymous visitors never reach
 * here; the route redirects them to /login.
 */
export function HomePage() {
  const me = useQuery(meQueryOptions)
  const characters = useQuery(charactersQueryOptions)
  const campaigns = useQuery(campaignsQueryOptions)
  const greetName = me.data?.name ?? me.data?.email ?? 'aventureiro'

  return (
    <PageChrome width="wide" className="space-y-8">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">
          Olá, {greetName}
        </h1>
        <p className="text-sm text-muted-foreground">Por onde começamos?</p>
      </header>

      <div className="grid gap-4 sm:grid-cols-3">
        <ShortcutTile
          to="/characters"
          Icon={Users2}
          label="Personagens"
          count={characters.data?.length}
        />
        <ShortcutTile
          to="/campaigns"
          Icon={Scroll}
          label="Campanhas"
          count={campaigns.data?.length}
        />
        <ShortcutTile to="/gm" Icon={Wand2} label="Ferramentas de mestre" />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <CharactersOverview
          characters={characters.data}
          loading={characters.isLoading}
        />
        <CampaignsOverview
          campaigns={campaigns.data}
          loading={campaigns.isLoading}
        />
      </div>
    </PageChrome>
  )
}

/** Big nav card into a main area, with an optional item count. */
function ShortcutTile({
  to,
  Icon,
  label,
  count,
}: {
  to: '/characters' | '/campaigns' | '/gm'
  Icon: LucideIcon
  label: string
  count?: number
}) {
  return (
    <Link to={to} className="group">
      <Card className="h-full transition-colors group-hover:border-primary">
        <CardContent className="flex items-center gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted">
            <Icon className="size-5" />
          </span>
          <div className="min-w-0">
            <p className="truncate font-medium">{label}</p>
            {count !== undefined && (
              <p className="text-sm text-muted-foreground">
                {count} {count === 1 ? 'item' : 'itens'}
              </p>
            )}
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}

/** "Personagens" overview column — recent characters + a create shortcut. */
function CharactersOverview({
  characters,
  loading,
}: {
  characters: Character[] | undefined
  loading: boolean
}) {
  return (
    <OverviewCard title="Personagens" viewAllTo="/characters">
      {loading ? (
        <SkeletonRows count={3} />
      ) : characters && characters.length > 0 ? (
        characters
          .slice(0, 5)
          .map((c) => (
            <OverviewRow
              key={c.id}
              to="/characters/$id"
              params={{ id: c.id }}
              label={c.name}
              trailing={<Badge variant="secondary">Nv {c.level}</Badge>}
            />
          ))
      ) : (
        <EmptyHint>Nenhum personagem ainda.</EmptyHint>
      )}
      <CreateRow to="/characters/new" label="Novo personagem" />
    </OverviewCard>
  )
}

/** "Campanhas" overview column — campaigns with the caller's role. */
function CampaignsOverview({
  campaigns,
  loading,
}: {
  campaigns: Campaign[] | undefined
  loading: boolean
}) {
  return (
    <OverviewCard title="Campanhas" viewAllTo="/campaigns">
      {loading ? (
        <SkeletonRows count={3} />
      ) : campaigns && campaigns.length > 0 ? (
        campaigns
          .slice(0, 5)
          .map((c) => (
            <OverviewRow
              key={c.id}
              to="/campaigns/$id"
              params={{ id: c.id }}
              label={c.name}
              trailing={
                c.role && (
                  <Badge variant="secondary">
                    {c.role === 'gm' ? 'Mestre' : 'Jogador'}
                  </Badge>
                )
              }
            />
          ))
      ) : (
        <EmptyHint>Nenhuma campanha ainda.</EmptyHint>
      )}
      <CreateRow to="/campaigns/new" label="Nova campanha" />
    </OverviewCard>
  )
}

/** Titled overview card with a "ver todos" link and a rows body. */
function OverviewCard({
  title,
  viewAllTo,
  children,
}: {
  title: string
  viewAllTo: '/characters' | '/campaigns'
  children: React.ReactNode
}) {
  return (
    <Card>
      <CardHeader className="flex items-center justify-between">
        <CardTitle>{title}</CardTitle>
        <Link
          to={viewAllTo}
          className="text-sm text-muted-foreground underline underline-offset-4"
        >
          Ver todos
        </Link>
      </CardHeader>
      <CardContent className="flex flex-col gap-1">{children}</CardContent>
    </Card>
  )
}

/** A single clickable overview row (character or campaign). */
function OverviewRow({
  to,
  params,
  label,
  trailing,
}: {
  to: '/characters/$id' | '/campaigns/$id'
  params: { id: number }
  label: string
  trailing?: React.ReactNode
}) {
  return (
    <Link
      to={to}
      params={params}
      className="flex items-center justify-between gap-2 rounded-md px-3 py-2 hover:bg-accent"
    >
      <span className="min-w-0 truncate">{label}</span>
      {trailing}
    </Link>
  )
}

/** Dashed "create new" row closing each overview column. */
function CreateRow({
  to,
  label,
}: {
  to: '/characters/new' | '/campaigns/new'
  label: string
}) {
  return (
    <Link
      to={to}
      className="flex items-center gap-2 rounded-md border border-dashed px-3 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-foreground"
    >
      <Plus className="size-4" />
      {label}
    </Link>
  )
}

function EmptyHint({ children }: { children: React.ReactNode }) {
  return <p className="px-3 py-2 text-sm text-muted-foreground">{children}</p>
}
