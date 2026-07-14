import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Plus, Search } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { HpBar } from '@/shared/ui/hp-bar'
import { Input } from '@/shared/ui/input'
import { MpBar } from '@/shared/ui/mp-bar'
import { PageChrome } from '@/shared/ui/page-chrome'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { cn } from '@/shared/lib/utils'
import type { Character } from '@/shared/api/api'
import {
  ATTRIBUTE_ABBR,
  ATTRIBUTE_KEYS,
} from '@/entities/character/expertise'
import { charactersQueryOptions } from '@/entities/character/queries'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import { CharacterPortrait } from '@/shared/ui/character-portrait'

// Headless table drives the roster search (name / class / origin) via the
// built-in global filter. Rows are read back out and rendered as thumbnails.
const columnHelper = createColumnHelper<Character>()
const columns = [
  columnHelper.accessor('name', { id: 'name' }),
  columnHelper.accessor((c) => primaryClass(c), { id: 'class' }),
  columnHelper.accessor('origin', { id: 'origin' }),
]

const EMPTY: Character[] = []

/**
 * Characters "select screen" — a Valorant-style agent-select in three
 * columns: a filterable thumbnail grid (left), the selected character's big
 * portrait + lock-in (middle), and its info panel (right). Phone stacks the
 * three. Selection is local UI state; "Abrir ficha" routes to the sheet.
 */
export function CharactersListPage() {
  const characters = useQuery(charactersQueryOptions)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [query, setQuery] = useState('')
  const roster = characters.data
  const selected = roster?.find((c) => c.id === selectedId) ?? roster?.[0]

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

  return (
    <PageChrome width="full" className="flex min-h-0 flex-1 flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Personagens</h1>
      </header>

      {characters.isLoading && <SkeletonCardGrid count={3} />}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {roster?.length === 0 && <NoCharacters />}

      {roster && roster.length > 0 && selected && filtered && (
        <div className="grid min-h-0 flex-1 gap-4 lg:grid-cols-[20rem_1fr_22rem]">
          <RosterPanel
            roster={filtered}
            query={query}
            onQuery={setQuery}
            selectedId={selected.id}
            onSelect={setSelectedId}
          />
          <SplashPanel character={selected} />
          <InfoPanel character={selected} />
        </div>
      )}
    </PageChrome>
  )
}

/** Left column: filter bar over a scrolling thumbnail grid + a create tile. */
function RosterPanel({
  roster,
  query,
  onQuery,
  selectedId,
  onSelect,
}: {
  roster: Character[]
  query: string
  onQuery: (q: string) => void
  selectedId: number
  onSelect: (id: number) => void
}) {
  return (
    <div className="flex min-h-0 min-w-0 flex-col gap-3">
      <div className="relative shrink-0">
        <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          value={query}
          onChange={(e) => onQuery(e.target.value)}
          placeholder="Buscar personagem"
          className="pl-8"
          aria-label="Buscar personagem"
        />
      </div>
      <div className="flex gap-2 overflow-x-auto pb-2 lg:grid lg:min-h-0 lg:flex-1 lg:auto-rows-min lg:grid-cols-3 lg:overflow-x-visible lg:overflow-y-auto lg:pb-0">
        {roster.map((c) => (
          <Thumb
            key={c.id}
            character={c}
            selected={c.id === selectedId}
            onSelect={() => onSelect(c.id)}
          />
        ))}
        <Link
          to="/characters/new"
          aria-label="Novo personagem"
          className="flex w-24 shrink-0 flex-col items-center justify-center gap-1 rounded-lg border border-dashed py-4 text-muted-foreground hover:bg-accent hover:text-foreground lg:aspect-square lg:w-auto lg:py-0"
        >
          <Plus className="size-5" />
          <span className="text-[11px]">Novo</span>
        </Link>
      </div>
    </div>
  )
}

/** A single portrait thumbnail in the roster grid. */
function Thumb({
  character,
  selected,
  onSelect,
}: {
  character: Character
  selected: boolean
  onSelect: () => void
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      title={character.name}
      className={cn(
        'flex w-24 shrink-0 flex-col items-center gap-1 rounded-lg border p-1.5 transition-colors lg:w-auto',
        selected ? 'border-primary bg-accent' : 'border-border hover:bg-accent',
      )}
    >
      <CharacterPortrait
        name={character.name}
        size="lg"
        className="aspect-square text-2xl"
      />
      <span className="w-full truncate text-center text-[11px]">
        {character.name}
      </span>
    </button>
  )
}

/** Middle column: dominant portrait with the name overlaid + a lock-in. */
function SplashPanel({ character }: { character: Character }) {
  return (
    <div className="flex min-h-0 min-w-0 flex-col gap-3">
      <div className="relative min-h-0 flex-1 overflow-hidden rounded-xl">
        <CharacterPortrait
          name={character.name}
          size="lg"
          className="aspect-auto h-full min-h-36 w-full rounded-xl lg:min-h-64"
        />
        <div className="absolute inset-x-0 bottom-0 flex items-end justify-between gap-2 bg-gradient-to-t from-background/90 to-transparent p-4">
          <div>
            <h2 className="text-3xl font-semibold tracking-tight">
              {character.name}
            </h2>
            <p className="text-sm text-muted-foreground">
              {primaryClass(character)}
            </p>
          </div>
          <Badge variant="secondary">Nv {character.level}</Badge>
        </div>
      </div>
      <Link
        to="/characters/$id"
        params={{ id: String(character.id) }}
        className="shrink-0"
      >
        <Button className="w-full" size="lg">
          Abrir ficha
        </Button>
      </Link>
    </div>
  )
}

/** Right column: role + name, vitals, and attributes for the selection. */
function InfoPanel({ character }: { character: Character }) {
  const races = character.races.map((r) => r.race).join(', ')

  return (
    <Card className="min-h-0 min-w-0 gap-0 overflow-y-auto py-4 lg:py-6">
      <CardContent className="flex flex-col gap-3 px-4 lg:gap-5 lg:px-6">
        <div>
          <p className="text-xs uppercase tracking-widest text-muted-foreground">
            {primaryClass(character)}
          </p>
          <h2 className="text-xl font-semibold tracking-tight lg:text-2xl">
            {character.name}
          </h2>
          <p className="text-sm text-muted-foreground">
            {[races, character.origin].filter(Boolean).join(' • ')}
          </p>
        </div>

        <div className="space-y-1.5">
          <HpBar current={character.hpCurrent} max={character.hpMax} size="sm" />
          <MpBar current={character.mpCurrent} max={character.mpMax} size="sm" />
        </div>

        <AttributeRow character={character} />
      </CardContent>
    </Card>
  )
}

/** Six-attribute mini grid (ABBR + signed modifier). */
function AttributeRow({ character }: { character: Character }) {
  return (
    <div className="grid grid-cols-6 gap-1.5 lg:grid-cols-2">
      {ATTRIBUTE_KEYS.map((key) => (
        <div
          key={key}
          className="flex flex-col items-center rounded-md border border-border px-1 py-1 lg:flex-row lg:justify-between lg:px-3 lg:py-1.5"
        >
          <span className="text-[10px] uppercase text-muted-foreground">
            {ATTRIBUTE_ABBR[key]}
          </span>
          <span className="font-mono text-sm font-semibold">
            {signed(character[key])}
          </span>
        </div>
      ))}
    </div>
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

function primaryClass(character: Character): string {
  const first = character.classes[0]
  if (!first) return character.origin
  return `${first.className} ${first.level}`
}

function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
}
