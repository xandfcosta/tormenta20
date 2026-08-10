import { Link, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  createColumnHelper,
  getCoreRowModel,
  getFilteredRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { Search } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { SkeletonCardGrid } from '@/shared/ui/skeleton'
import type { Character } from '@/shared/api/api'
import { charactersQueryOptions } from '@/entities/character/queries'
import { SceneShell } from '@/shared/layout/scene-shell'
import { fuzzyFilter } from '@/shared/lib/fuzzy-filter'
import { useSceneNav } from '@/shared/lib/use-scene-nav'
import { useSfx } from '@/shared/lib/use-sfx'
import { CharacterFilmstrip } from '@/features/character-select/character-filmstrip'
import { CharacterStage } from '@/features/character-select/character-stage'
import { DossierDrawer } from '@/features/character-select/dossier-drawer'

// Headless table drives the roster search. Indexed columns: name, primary
// class, origin, and races.
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
 * Characters select screen — design "palco + dossiê": the selected character
 * on a spotlit center stage with prev/next peeking from the sides, a
 * filmstrip index for O(1) jumps across long rosters, and a dossier drawer
 * for the readable detail. Search filters the roster; the stage snaps to the
 * first match. `←/→` navigate, Enter opens the sheet, D toggles the dossier,
 * `/` focuses search.
 */
export function CharactersListPage() {
  const characters = useQuery(charactersQueryOptions)
  const navigate = useNavigate()
  const sfx = useSfx()
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [dossierOpen, setDossierOpen] = useState(false)
  const [direction, setDirection] = useState<1 | -1>(1)
  const [query, setQuery] = useState('')
  const roster = characters.data

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
  const index = Math.max(
    0,
    filtered.findIndex((c) => c.id === selectedId),
  )
  const selected = filtered[index] ?? null
  const prev = index > 0 ? filtered[index - 1] : null
  const next = index < filtered.length - 1 ? filtered[index + 1] : null

  const step = (delta: number) => {
    if (filtered.length === 0) return
    sfx('hover')
    const nextIndex = Math.min(filtered.length - 1, Math.max(0, index + delta))
    setDirection(delta >= 0 ? 1 : -1)
    setSelectedId(filtered[nextIndex].id)
  }
  const jumpTo = (id: number) => {
    const target = filtered.findIndex((c) => c.id === id)
    if (target === -1) return
    sfx('select')
    setDirection(target >= index ? 1 : -1)
    setSelectedId(id)
  }
  const openSheet = () => {
    if (!selected) return
    sfx('select')
    navigate({ to: '/characters/$id', params: { id: selected.id } })
  }

  // The roster is a selection scene (the spotlight stage): a `delegated`
  // scene-nav so it shares the grammar + gamepad seam while keeping its own
  // cursor. onCommand maps the standard grammar; onKey holds the keyboard-only
  // accelerators (D dossier, / search) and search Esc-to-clear.
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
          if (cmd.dir === 'left') {
            step(-1)
            return true
          }
          if (cmd.dir === 'right') {
            step(1)
            return true
          }
          return false // ↑/↓ have no meaning on the horizontal stage
        case 'edge':
          if (filtered.length > 0) {
            jumpTo(filtered[cmd.to === 'first' ? 0 : filtered.length - 1].id)
          }
          return true
        case 'bumper':
          step(cmd.dir === 'next' ? 5 : -5)
          return true
        case 'activate':
          openSheet()
          return true
        case 'back':
          if (dossierOpen) {
            setDossierOpen(false)
            return true
          }
          if (query.length > 0) {
            setQuery('')
            return true
          }
          return false // nothing to close → onEscape leaves the scene
      }
    },
    onKey: (e) => {
      const el = e.target as HTMLElement
      if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return false // let the field handle its keys
        setQuery('')
        el.blur()
        return true
      }
      if (e.key === 'd' || e.key === 'D') {
        setDossierOpen((v) => !v)
        return true
      }
      if (e.key === '/') {
        e.preventDefault()
        document.querySelector<HTMLInputElement>('[data-roster-search]')?.focus()
        return true
      }
      return false
    },
  })

  const headerControls =
    roster && roster.length > 0 ? (
      <>
        <div className="relative min-w-40 flex-1 sm:max-w-xs">
          <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Buscar nome, classe, origem, raça"
            className="pl-8"
            aria-label="Buscar personagem"
            data-roster-search
          />
        </div>
        <Badge variant="secondary" className="shrink-0">
          {filtered.length} de {roster.length}
        </Badge>
        <Link to="/characters/new">
          <Button size="sm">+ Novo</Button>
        </Link>
      </>
    ) : null

  return (
    <SceneShell
      dense
      title="Personagens"
      onBack={() => {
        sfx('select')
        navigate({ to: '/' })
      }}
      onEnter={() => sfx('transition')}
      headerRight={headerControls}
      className="gap-2"
    >
      {characters.isLoading && <SkeletonCardGrid count={3} />}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {roster?.length === 0 && <EmptyStage />}
      {roster && roster.length > 0 && filtered.length === 0 && (
        <NoMatches query={query} onClear={() => setQuery('')} />
      )}

      {selected && (
        <div className="relative flex min-h-0 flex-1 flex-col">
          <CharacterStage
            selected={selected}
            prev={prev}
            next={next}
            direction={direction}
            onStep={step}
            onOpen={openSheet}
            onDossier={() => setDossierOpen((v) => !v)}
            dossierOpen={dossierOpen}
          />
          <DossierDrawer
            character={selected}
            open={dossierOpen}
            onClose={() => setDossierOpen(false)}
          />
          <CharacterFilmstrip
            roster={filtered}
            selectedId={selected.id}
            onSelect={jumpTo}
            onHover={() => sfx('hover')}
          />
          {/* Keyboard hints only where there's a keyboard: laptop+desktop (≥xl).
              Hidden on tablet/phone (touch) — the keys don't apply there. */}
          <p className="hidden pt-1 text-center text-[11px] text-muted-foreground xl:block">
            ← → navegar · Enter abrir ficha · D dossiê · / buscar
          </p>
        </div>
      )}
    </SceneShell>
  )
}

/** Empty roster: the stage itself invites the first character. */
function EmptyStage() {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4">
      <div className="flex aspect-[3/4] w-48 items-center justify-center rounded-md border-2 border-dashed border-grimorio-iron">
        <span className="select-none font-heading text-7xl text-grimorio-gold/30">
          ?
        </span>
      </div>
      <p className="font-heading text-xl uppercase tracking-[0.12em] text-foreground">
        Seu grupo aguarda um herói
      </p>
      <Link to="/characters/new">
        <Button size="lg">Criar seu primeiro personagem</Button>
      </Link>
    </div>
  )
}

function NoMatches({ query, onClear }: { query: string; onClear: () => void }) {
  return (
    <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3">
      <p className="font-heading text-sm uppercase tracking-widest text-grimorio-gold/70">
        Nada encontrado
      </p>
      <p className="text-sm text-muted-foreground">
        Nenhum personagem para “{query}”.
      </p>
      <Button variant="outline" onClick={onClear}>
        Limpar busca<span className="hidden xl:inline"> (Esc)</span>
      </Button>
    </div>
  )
}
