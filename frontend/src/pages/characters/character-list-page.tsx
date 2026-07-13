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
        <div className="grid gap-6 lg:grid-cols-[18rem_1fr]">
          <CharacterRoster
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

/** Left column: selectable list of portrait + name rows, then a create row. */
function CharacterRoster({
  roster,
  selectedId,
  onSelect,
}: {
  roster: Character[]
  selectedId: number
  onSelect: (id: number) => void
}) {
  return (
    <div className="order-2 flex flex-col gap-2 lg:order-1">
      {roster.map((c) => (
        <RosterRow
          key={c.id}
          character={c}
          selected={c.id === selectedId}
          onSelect={() => onSelect(c.id)}
        />
      ))}
      <Link
        to="/characters/new"
        className="flex items-center gap-2 rounded-lg border border-dashed px-3 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <Plus className="size-4" />
        Novo personagem
      </Link>
    </div>
  )
}

function RosterRow({
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
        'flex items-center gap-3 rounded-lg border p-2 text-left transition-colors',
        selected
          ? 'border-primary bg-accent'
          : 'border-border hover:bg-accent',
      )}
    >
      <CharacterPortrait name={character.name} size="sm" />
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium">{character.name}</p>
        <p className="truncate text-xs text-muted-foreground">
          {primaryClass(character)}
        </p>
      </div>
      <Badge variant="secondary">Nv {character.level}</Badge>
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
    <Card className="order-1 lg:order-2">
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

function primaryClass(character: Character): string {
  const first = character.classes[0]
  if (!first) return character.origin
  return `${first.className} ${first.level}`
}

function signed(n: number): string {
  return n >= 0 ? `+${n}` : String(n)
}
