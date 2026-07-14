import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  type ColumnFiltersState,
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { CalendarClock, Plus, Search, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/shared/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { cn } from '@/shared/lib/utils'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import { CharacterPortrait } from '@/shared/ui/character-portrait'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import type { Campaign } from '@/shared/api/api'

type RoleFilter = 'all' | 'gm' | 'player'

const ROLE_FILTERS: { value: RoleFilter; label: string }[] = [
  { value: 'all', label: 'Todas' },
  { value: 'gm', label: 'Mestrando' },
  { value: 'player', label: 'Jogando' },
]

// Headless table columns: the accessors feed the built-in global (search)
// and column (role) filters. Nothing here renders — we read the filtered
// rows back out and render them as grouped cards.
const columnHelper = createColumnHelper<Campaign>()
const columns = [
  columnHelper.accessor('name', { id: 'name' }),
  columnHelper.accessor((c) => c.description ?? '', { id: 'description' }),
  columnHelper.accessor((c) => c.role ?? 'player', {
    id: 'role',
    filterFn: 'equals',
  }),
]

const EMPTY: Campaign[] = []

/**
 * Campaigns list, grouped by the caller's role ("Mestrando" / "Jogando").
 * Filtering (name/description search + role) runs through a headless
 * TanStack Table; the filtered rows are rendered as cards, not a table.
 */
export function CampaignsListPage() {
  const campaigns = useQuery(campaignsQueryOptions)
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])

  const table = useReactTable({
    data: campaigns.data ?? EMPTY,
    columns,
    state: { globalFilter, columnFilters },
    onGlobalFilterChange: setGlobalFilter,
    onColumnFiltersChange: setColumnFilters,
    globalFilterFn: fuzzyFilter<Campaign>(),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })

  const visible = table.getRowModel().rows.map((r) => r.original)
  const mastering = visible.filter((c) => c.role === 'gm')
  const playing = visible.filter((c) => c.role !== 'gm')
  const hasData = (campaigns.data?.length ?? 0) > 0

  const role =
    (columnFilters.find((f) => f.id === 'role')?.value as RoleFilter) ?? 'all'
  const setRole = (r: RoleFilter) =>
    setColumnFilters(r === 'all' ? [] : [{ id: 'role', value: r }])

  return (
    <PageChrome className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">Campanhas</h1>
        <div className="flex gap-2">
          <Link to="/campaigns/join">
            <Button variant="outline">
              <UserPlus className="mr-1 size-4" /> Entrar em campanha
            </Button>
          </Link>
          <Link to="/campaigns/new">
            <Button>
              <Plus className="mr-1 size-4" /> Nova campanha
            </Button>
          </Link>
        </div>
      </div>

      {hasData && (
        <CampaignFilters
          query={globalFilter}
          onQuery={setGlobalFilter}
          role={role}
          onRole={setRole}
        />
      )}

      {campaigns.isLoading && <SkeletonCardGrid count={3} />}
      {campaigns.isError && (
        <p className="text-destructive">{(campaigns.error as Error).message}</p>
      )}
      {campaigns.data?.length === 0 && <NoCampaigns />}

      {mastering.length > 0 && (
        <CampaignSection title="Mestrando" campaigns={mastering} />
      )}
      {playing.length > 0 && (
        <CampaignSection title="Jogando" campaigns={playing} />
      )}
      {hasData && visible.length === 0 && (
        <p className="py-6 text-center text-sm text-muted-foreground">
          Nenhuma campanha corresponde ao filtro.
        </p>
      )}
    </PageChrome>
  )
}

/** Search box + role segmented filter over the campaign list. */
function CampaignFilters({
  query,
  onQuery,
  role,
  onRole,
}: {
  query: string
  onQuery: (q: string) => void
  role: RoleFilter
  onRole: (r: RoleFilter) => void
}) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
      <div className="relative sm:max-w-xs sm:flex-1">
        <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
          placeholder="Buscar campanha"
          className="pl-8"
          aria-label="Buscar campanha"
        />
      </div>
      <div className="flex gap-1">
        {ROLE_FILTERS.map((f) => (
          <Button
            key={f.value}
            type="button"
            size="sm"
            variant={role === f.value ? 'default' : 'outline'}
            aria-pressed={role === f.value}
            onClick={() => onRole(f.value)}
            className={cn(role === f.value && 'pointer-events-none')}
          >
            {f.label}
          </Button>
        ))}
      </div>
    </div>
  )
}

/** A titled role section with a responsive card grid. */
function CampaignSection({
  title,
  campaigns,
}: {
  title: string
  campaigns: Campaign[]
}) {
  return (
    <section className="space-y-3">
      <h2 className="text-sm font-medium uppercase tracking-widest text-muted-foreground">
        {title}
      </h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {campaigns.map((c) => (
          <CampaignCard key={c.id} campaign={c} />
        ))}
      </div>
    </section>
  )
}

function CampaignCard({ campaign }: { campaign: Campaign }) {
  return (
    <Link to="/campaigns/$id" params={{ id: campaign.id }}>
      <Card className="h-full transition-colors hover:border-primary">
        <CardHeader>
          <CardTitle>{campaign.name}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          {campaign.description && (
            <p className="line-clamp-3 text-muted-foreground">
              {campaign.description}
            </p>
          )}
          {campaign.character && <MyCharacterRow character={campaign.character} />}
          <p className="flex items-center gap-1 text-xs text-muted-foreground">
            <CalendarClock className="size-3" />
            Atualizada em{' '}
            {new Date(campaign.updatedAt).toLocaleDateString('pt-BR')}
          </p>
        </CardContent>
      </Card>
    </Link>
  )
}

/** The caller's PC in a campaign — portrait + name + class/level line. */
function MyCharacterRow({
  character,
}: {
  character: NonNullable<Campaign['character']>
}) {
  const classes = character.classes
    .map((c) => `${c.className} ${c.level}`)
    .join(' / ')
  return (
    <div className="flex items-center gap-2 rounded-md border p-2">
      <CharacterPortrait name={character.name} size="sm" />
      <div className="min-w-0">
        <p className="truncate font-medium">{character.name}</p>
        <p className="truncate text-xs text-muted-foreground">
          {classes || `Nv ${character.level}`}
        </p>
      </div>
    </div>
  )
}

function NoCampaigns() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
        <p>Nenhuma campanha ainda.</p>
        <Link to="/campaigns/new">
          <Button>Criar sua primeira campanha</Button>
        </Link>
      </CardContent>
    </Card>
  )
}
