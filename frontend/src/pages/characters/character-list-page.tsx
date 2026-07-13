import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardContent } from '@/shared/ui/card'
import { HpBar } from '@/shared/ui/hp-bar'
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
import { CharacterPortrait } from './character-portrait'

/**
 * Characters "select screen" — a game-style master/detail: a roster of
 * portraits + names on the left, a big portrait + stat panel for the
 * selected character on the right. Phone stacks (detail on top, roster
 * below). Selection is local UI state; opening the sheet routes to
 * /characters/$id.
 */
export function CharactersListPage() {
  const characters = useQuery(charactersQueryOptions)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const roster = characters.data
  const selected = roster?.find((c) => c.id === selectedId) ?? roster?.[0]

  return (
    <PageChrome className="space-y-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">Personagens</h1>
      </header>

      {characters.isLoading && <SkeletonCardGrid count={3} />}
      {characters.isError && (
        <p className="text-destructive">{(characters.error as Error).message}</p>
      )}
      {roster?.length === 0 && <NoCharacters />}

      {roster && roster.length > 0 && selected && (
        <div className="flex flex-col-reverse gap-4 lg:flex-row lg:gap-6">
          <AgentRail
            roster={roster}
            selectedId={selected.id}
            onSelect={setSelectedId}
          />
          <CharacterDetail character={selected} />
        </div>
      )}
    </PageChrome>
  )
}

/**
 * Agent-select rail (Valorant-style): a scrolling strip of portrait
 * thumbnails. Desktop → narrow vertical column on the left; phone →
 * horizontal strip along the bottom (parent uses flex-col-reverse). A
 * trailing "+" thumbnail creates a new character.
 */
function AgentRail({
  roster,
  selectedId,
  onSelect,
}: {
  roster: Character[]
  selectedId: number
  onSelect: (id: number) => void
}) {
  return (
    <div className="flex shrink-0 gap-2 overflow-x-auto pb-2 lg:w-28 lg:flex-col lg:overflow-x-visible lg:overflow-y-auto lg:pb-0">
      {roster.map((c) => (
        <RailThumb
          key={c.id}
          character={c}
          selected={c.id === selectedId}
          onSelect={() => onSelect(c.id)}
        />
      ))}
      <Link
        to="/characters/new"
        aria-label="Novo personagem"
        className="flex w-20 shrink-0 flex-col items-center justify-center gap-1 rounded-lg border border-dashed py-2 text-muted-foreground hover:bg-accent hover:text-foreground lg:w-full"
      >
        <Plus className="size-5" />
        <span className="text-[11px]">Novo</span>
      </Link>
    </div>
  )
}

/** A single portrait thumbnail in the rail. */
function RailThumb({
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
      className={cn(
        'flex w-20 shrink-0 flex-col items-center gap-1 rounded-lg border p-1.5 transition-colors lg:w-full',
        selected ? 'border-primary bg-accent' : 'border-border hover:bg-accent',
      )}
    >
      <CharacterPortrait name={character.name} size="sm" />
      <span className="w-full truncate text-center text-[11px]">
        {character.name}
      </span>
    </button>
  )
}

/** Right column: big portrait + stat panel for the selected character. */
function CharacterDetail({ character }: { character: Character }) {
  const races = character.races.map((r) => r.race).join(', ')
  const classes = character.classes
    .map((c) => `${c.className} ${c.level}`)
    .join(' / ')

  return (
    <Card className="min-w-0 flex-1">
      <CardContent className="grid gap-6 sm:grid-cols-[16rem_1fr]">
        <CharacterPortrait name={character.name} size="lg" />

        <div className="flex flex-col gap-4">
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-2xl font-semibold tracking-tight">
                {character.name}
              </h2>
              <Badge variant="secondary">Nv {character.level}</Badge>
            </div>
            <p className="text-sm text-muted-foreground">
              {[races, character.origin].filter(Boolean).join(' • ')}
            </p>
            {classes && <p className="text-sm font-medium">{classes}</p>}
          </div>

          <div className="space-y-1.5">
            <HpBar current={character.hpCurrent} max={character.hpMax} size="sm" />
            <MpBar current={character.mpCurrent} max={character.mpMax} size="sm" />
          </div>

          <AttributeRow character={character} />

          <div className="mt-auto flex flex-wrap gap-2 pt-2">
            <Link to="/characters/$id" params={{ id: String(character.id) }}>
              <Button>Abrir ficha</Button>
            </Link>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

/** Six-attribute mini grid (ABBR + signed modifier). */
function AttributeRow({ character }: { character: Character }) {
  return (
    <div className="grid grid-cols-6 gap-1.5">
      {ATTRIBUTE_KEYS.map((key) => (
        <div
          key={key}
          className="flex flex-col items-center rounded-md border border-border py-1.5"
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

function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
}
