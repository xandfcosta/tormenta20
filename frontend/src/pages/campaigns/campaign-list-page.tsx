import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  type ColumnFiltersState,
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Plus, Search, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import { SceneShell } from '@/shared/layout/scene-shell'
import { cn } from '@/shared/lib/utils'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import { useSceneNav } from '@/shared/lib/use-scene-nav'
import { useSfx } from '@/shared/lib/use-sfx'
import { campaignsQueryOptions } from '@/entities/campaign/queries'
import type { Campaign } from '@/shared/api/api'
import { CampaignRail } from '@/features/campaign-select/campaign-rail'
import { CampaignBook } from '@/features/campaign-select/campaign-book'
import { useActiveSessionByCampaign } from '@/features/campaign-select/use-active-sessions'

type RoleFilter = 'all' | 'gm' | 'player'

const ROLE_FILTERS: { value: RoleFilter; label: string }[] = [
  { value: 'all', label: 'Todas' },
  { value: 'gm', label: 'Mestrando' },
  { value: 'player', label: 'Jogando' },
]

// Headless table columns: the accessors feed the built-in global (search)
// and column (role) filters. Nothing renders from here — we read the filtered
// rows back out and drive the rail/stage with them.
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
 * Crônicas — the campaigns roster as a cinematic "chapter select" scene
 * (ALE-56): the focused chronicle on a stage (emblem, synopsis, your PC, live
 * status, primary action) with a rail to switch focus, mirroring the character
 * selector. Search + role filter run through a headless TanStack Table;
 * `←/→` (or `↑/↓`) move focus, Enter opens (resumes a live session or opens the
 * chronicle), `/` focuses search, Esc clears.
 */
export function CampaignsListPage() {
  const campaigns = useQuery(campaignsQueryOptions)
  const navigate = useNavigate()
  const sfx = useSfx()
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [globalFilter, setGlobalFilter] = useState('')
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([])
  const roster = campaigns.data

  const table = useReactTable({
    data: roster ?? EMPTY,
    columns,
    state: { globalFilter, columnFilters },
    onGlobalFilterChange: setGlobalFilter,
    onColumnFiltersChange: setColumnFilters,
    globalFilterFn: fuzzyFilter<Campaign>(),
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
  })
  const filtered = table.getRowModel().rows.map((r) => r.original)
  const index = Math.max(
    0,
    filtered.findIndex((c) => c.id === selectedId),
  )
  const selected = filtered[index] ?? null
  const activeByCampaign = useActiveSessionByCampaign(
    (roster ?? EMPTY).map((c) => c.id),
  )
  const hasData = (roster?.length ?? 0) > 0

  // Every pick feeds the book; it queues them and turns to each in order (see
  // usePageTurns). `orderIds` gives each turn its direction.
  const orderIds = filtered.map((c) => c.id)

  const role =
    (columnFilters.find((f) => f.id === 'role')?.value as RoleFilter) ?? 'all'
  const setRole = (r: RoleFilter) =>
    setColumnFilters(r === 'all' ? [] : [{ id: 'role', value: r }])

  const step = (delta: number) => {
    if (filtered.length === 0) return
    sfx('hover')
    const next = Math.min(filtered.length - 1, Math.max(0, index + delta))
    setSelectedId(filtered[next].id)
  }
  const jumpTo = (id: number) => {
    sfx('select')
    setSelectedId(id)
  }
  const openDetail = (c: Campaign) => {
    sfx('select')
    navigate({ to: '/campaigns/$id', params: { id: c.id } })
  }
  const resume = (c: Campaign) => {
    const sid = activeByCampaign[c.id]
    if (sid == null) return openDetail(c)
    sfx('select')
    navigate({
      to: '/campaigns/$id/sessions/$sid',
      params: { id: c.id, sid },
    })
  }
  const openFocused = () => {
    if (!selected) return
    if (activeByCampaign[selected.id] != null) resume(selected)
    else openDetail(selected)
  }

  // The chronicles list is a selection scene (the 1-D book rail): a `delegated`
  // scene-nav so it shares the grammar + gamepad seam while keeping its own
  // cursor. onKey holds the keyboard-only accelerators (/ search, O open).
  const leaveScene = () => {
    sfx('back')
    navigate({ to: '/' })
  }
  useSceneNav({
    root: () => document.querySelector<HTMLElement>('[data-slot="scene-shell"]'),
    delegated: true,
    sfx,
    onEscape: leaveScene,
    onCommand: (cmd) => {
      switch (cmd.type) {
        case 'move':
          // A 1-D rail: both axes step (←/↑ back, →/↓ forward).
          step(cmd.dir === 'right' || cmd.dir === 'down' ? 1 : -1)
          return true
        case 'edge':
          if (filtered.length > 0) {
            jumpTo(filtered[cmd.to === 'first' ? 0 : filtered.length - 1].id)
          }
          return true
        case 'bumper':
          step(cmd.dir === 'next' ? 5 : -5)
          return true
        case 'activate':
          openFocused()
          return true
        case 'back':
          if (globalFilter.length > 0) {
            setGlobalFilter('')
            return true
          }
          return false // nothing to clear → onEscape leaves the scene
      }
    },
    onKey: (e) => {
      const el = e.target as HTMLElement
      if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return false
        setGlobalFilter('')
        el.blur()
        return true
      }
      if (e.key === 'o' || e.key === 'O') {
        if (selected) openDetail(selected)
        return true
      }
      if (e.key === '/') {
        e.preventDefault()
        document
          .querySelector<HTMLInputElement>('[data-campaign-search]')
          ?.focus()
        return true
      }
      return false
    },
  })

  const headerControls = (
    <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto sm:flex-1 sm:justify-end">
      {hasData && (
        <div className="relative w-full sm:w-56 md:w-64">
          <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            placeholder="Buscar campanha"
            className="pl-8"
            aria-label="Buscar campanha"
            data-campaign-search
          />
        </div>
      )}
      <div className="flex flex-1 flex-wrap items-center gap-2 sm:flex-none">
        {hasData && <RoleFilterButtons role={role} onRole={setRole} />}
        <div className="ml-auto flex gap-2 sm:ml-0">
          <Link to="/campaigns/join">
            <Button variant="outline" size="sm" aria-label="Entrar em campanha">
              <UserPlus className="size-4 sm:mr-1" />
              <span className="hidden sm:inline">Entrar</span>
            </Button>
          </Link>
          <Link to="/campaigns/new">
            <Button size="sm" aria-label="Nova campanha">
              <Plus className="size-4 sm:mr-1" />
              <span className="hidden sm:inline">Nova</span>
            </Button>
          </Link>
        </div>
      </div>
    </div>
  )

  return (
    <SceneShell
      dense
      title="Crônicas"
      onBack={() => {
        sfx('select')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={headerControls}
    >
      {campaigns.isLoading && <SkeletonCardGrid count={3} />}
      {campaigns.isError && (
        <p className="text-destructive">{(campaigns.error as Error).message}</p>
      )}
      {roster?.length === 0 && <NoCampaigns />}
      {hasData && filtered.length === 0 && (
        <NoMatches onClear={() => setGlobalFilter('')} />
      )}

      {selected && (
        <div className="flex min-h-0 flex-1 items-center justify-center">
          <div className="flex min-h-0 w-full max-w-7xl flex-col gap-2 lg:flex-row lg:items-stretch lg:gap-0">
            {/* Mirrors the bookmarks' width so the book itself centers on the
                viewport (not the book+tabs group). Hidden on phones. */}
            <div aria-hidden className="hidden lg:block lg:w-56 lg:shrink-0" />
            <CampaignBook
              campaign={selected}
              isLive={activeByCampaign[selected.id] != null}
              orderIds={orderIds}
              onOpen={() => openDetail(selected)}
              onResume={() => resume(selected)}
            />
            <CampaignRail
              campaigns={filtered}
              selectedId={selected.id}
              activeByCampaign={activeByCampaign}
              onSelect={jumpTo}
              onHover={() => sfx('hover')}
              className="lg:-ml-px lg:w-56 lg:shrink-0 lg:self-stretch"
            />
          </div>
        </div>
      )}
    </SceneShell>
  )
}

/** Role segmented filter (Todas / Mestrando / Jogando). */
function RoleFilterButtons({
  role,
  onRole,
}: {
  role: RoleFilter
  onRole: (r: RoleFilter) => void
}) {
  return (
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
  )
}

/** Empty roster: theatrical CTA matching the grimório scene. */
function NoCampaigns() {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 py-10">
      <div className="flex size-24 items-center justify-center rounded-md border-2 border-dashed border-grimorio-iron">
        <span className="select-none font-heading text-5xl text-grimorio-gold/30">
          ✦
        </span>
      </div>
      <p className="font-heading text-xl uppercase tracking-[0.12em] text-foreground">
        Nenhuma crônica ainda
      </p>
      <Link to="/campaigns/new">
        <Button size="lg">Criar sua primeira campanha</Button>
      </Link>
    </div>
  )
}

function NoMatches({ onClear }: { onClear: () => void }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 py-10">
      <p className="font-heading text-sm uppercase tracking-widest text-grimorio-gold/70">
        Nada encontrado
      </p>
      <p className="text-sm text-muted-foreground">
        Nenhuma campanha corresponde ao filtro.
      </p>
      <Button variant="outline" onClick={onClear}>
        Limpar busca<span className="hidden xl:inline"> (Esc)</span>
      </Button>
    </div>
  )
}
