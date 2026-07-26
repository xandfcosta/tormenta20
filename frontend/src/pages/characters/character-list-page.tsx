import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { LayoutGrid, Rows3, Search } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import type { Character } from '@/shared/api/api'
import { charactersQueryOptions } from '@/entities/character/queries'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import {
  CharacterInfoPanel,
  CharacterSummaryBar,
} from '@/features/character-select/character-info-panel'
import { CharacterSplash } from '@/features/character-select/character-splash'
import {
  NewCharacterTile,
  Roster,
  RosterHint,
} from '@/features/character-select/roster-strip'

// Past this many characters the roster defaults to the full grid (better scan
// surface than a long horizontal strip) until the user toggles manually.
const AUTO_EXPAND_THRESHOLD = 12

// Headless table drives the roster search. Indexed columns: name, primary
// class, origin, and races (races are displayed in the panel, so they must be
// searchable too).
const columnHelper = createColumnHelper<Character>()
const columns = [
  columnHelper.accessor('name', { id: 'name' }),
  columnHelper.accessor((c) => c.classes[0]?.className ?? '', { id: 'class' }),
  columnHelper.accessor('origin', { id: 'origin' }),
  columnHelper.accessor((c) => c.races.map((r) => r.race).join(' '), {
    id: 'races',
  }),
]

const EMPTY: Character[] = []

/**
 * Characters "select screen" — a Valorant-style agent-select: a dominant
 * character splash (left) + an info panel (right) over a 2-row roster strip
 * (bottom). Search is pinned to the header so it never scrolls with the strip;
 * an "Expandir" toggle pops the roster into a full grid for large rosters.
 * Selection is local UI state; "Abrir ficha" routes to the sheet.
 */
export function CharactersListPage() {
  const characters = useQuery(charactersQueryOptions)
  const navigate = useNavigate()
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [query, setQuery] = useState('')
  // null = follow the auto rule (grid for big rosters); a bool = manual override.
  const [expandOverride, setExpandOverride] = useState<boolean | null>(null)
  const roster = characters.data
  const selected = roster?.find((c) => c.id === selectedId) ?? roster?.[0]
  const expanded =
    expandOverride ?? (roster?.length ?? 0) > AUTO_EXPAND_THRESHOLD

  const table = useReactTable({
    data: roster ?? EMPTY,
    columns,
    state: { globalFilter: query },
    onGlobalFilterChange: setQuery,
    globalFilterFn: fuzzyFilter<Character>(),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })
  const filtered = table.getRowModel().rows.map((r) => r.original)

  const openSheet = (id: number) =>
    navigate({ to: '/characters/$id', params: { id } })

  return (
    <PageChrome width="full" className="flex min-h-0 flex-1 flex-col gap-4">
      <header className="flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">Personagens</h1>
        {roster && roster.length > 0 && (
          <div className="ml-auto flex flex-1 flex-wrap items-center justify-end gap-2">
            <div className="relative min-w-48 flex-1 sm:max-w-xs">
              <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="search"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Buscar nome, classe, origem, raça"
                className="pl-8"
                aria-label="Buscar personagem"
              />
            </div>
            <Badge variant="secondary" className="shrink-0">
              {filtered.length} de {roster.length}
            </Badge>
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-pressed={expanded}
              onClick={() => setExpandOverride(!expanded)}
            >
              {expanded ? (
                <Rows3 className="size-4" />
              ) : (
                <LayoutGrid className="size-4" />
              )}
              {expanded ? 'Recolher' : 'Ver todos'}
            </Button>
            <NewCharacterTile />
          </div>
        )}
      </header>

      {characters.isLoading && <SkeletonCardGrid count={3} />}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {roster?.length === 0 && <NoCharacters />}

      {roster && roster.length > 0 && selected && (
        <div className="flex min-h-0 flex-1 flex-col gap-3">
          {expanded ? (
            <>
              <CharacterSummaryBar character={selected} />
              <div className="min-h-0 flex-1 overflow-y-auto">
                <Roster
                  roster={filtered}
                  selectedId={selected.id}
                  onSelect={setSelectedId}
                  onOpen={openSheet}
                  expanded
                />
              </div>
              <RosterHint />
            </>
          ) : (
            <>
              <div className="grid min-h-0 flex-1 gap-4 lg:grid-cols-[1fr_22rem]">
                <CharacterSplash character={selected} />
                <CharacterInfoPanel character={selected} />
              </div>
              <Roster
                roster={filtered}
                selectedId={selected.id}
                onSelect={setSelectedId}
                onOpen={openSheet}
                expanded={false}
              />
              <RosterHint />
            </>
          )}
        </div>
      )}
    </PageChrome>
  )
}

function NoCharacters() {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 py-10 text-muted-foreground">
        <p>Nenhum personagem ainda.</p>
        <Link to="/characters/new">
          <Button>Criar seu primeiro personagem</Button>
        </Link>
      </CardContent>
    </Card>
  )
}
