import { Settings2 } from 'lucide-react'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import { ownedAbilities, type SheetSearchEntry } from './sheet-search-index'
import { type ReactNode, useMemo, useState } from 'react'
import { getRace } from '@/shared/lib/abilities-cache'
import type { RaceDefinition } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import type { ActivationSpec } from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import { resolveActivationSpec } from '@/entities/character/use-power-action'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import type { CardFocus } from './collapsible-ability-card'
import { ClassesSection } from './class-abilities'
import { PowerPlayList } from './power-play-list'
import { normalize } from './normalize'
import { OriginAbilitySection } from './origin-abilities'
import { RaceAbilitySection } from './race-abilities'
import {
  computePendencias,
  type Pendencia,
  type PendenciaSource,
} from './pendencias'
import { PendenciasCallout } from './pendencias-callout'
import { PowerActionSlot } from './power-action-slot'

/**
 * "Habilidades" tab — a Pendências callout over three source sub-tabs
 * (Raça / Origem / Classe). Each source renders its abilities as collapsible
 * cards; clicking a pendência jumps to its tab and opens the owning card.
 */
export function AbilitiesPanel({ character }: { character: Character }) {
  const races = character.races
    .map((r) => getRace(r.race))
    .filter((r): r is RaceDefinition => Boolean(r))
  const classes = character.classes

  const pendencias = useMemo(() => computePendencias(character), [character])
  const pendingByCard = useMemo(() => {
    const m = new Map<string, number>()
    for (const p of pendencias) m.set(p.cardId, (m.get(p.cardId) ?? 0) + 1)
    return m
  }, [pendencias])

  // Open on the first source that still owes a choice; falls back to Raça.
  const [tab, setTab] = useState<PendenciaSource>(
    pendencias[0]?.source ?? 'raca',
  )
  const [focus, setFocus] = useState<CardFocus>(null)
  const [query, setQuery] = useState('')
  // Play list by default; acquisition UI (checkboxes, pendências) lives
  // behind "Editar poderes". Open in edit when choices are still owed so
  // onboarding isn't hidden.
  const [mode, setMode] = useState<'play' | 'edit'>(
    pendencias.length > 0 ? 'edit' : 'play',
  )

  const jump = (p: Pendencia) => {
    setMode('edit')
    setTab(p.source)
    setFocus({ id: p.cardId, nonce: Date.now() })
  }

  const countFor = (source: PendenciaSource) =>
    pendencias.filter((p) => p.source === source).length

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
      {mode === 'edit' && (
        <PendenciasCallout pendencias={pendencias} onJump={jump} />
      )}

      {/* Flat lookup by NAME (audit: at the table the player knows the power's
          name, not which source granted it). A non-empty query replaces the
          list with one filtered result set + source badges. */}
      <div className="flex shrink-0 items-center gap-2">
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Buscar poder ou habilidade…"
          aria-label="Buscar poder ou habilidade"
          className="h-8 min-w-0 flex-1 text-xs"
        />
        <Button
          type="button"
          variant={mode === 'edit' ? 'default' : 'outline'}
          size="sm"
          className="h-8 shrink-0 gap-1 text-xs"
          onClick={() => setMode(mode === 'edit' ? 'play' : 'edit')}
        >
          <Settings2 className="size-3.5" />
          {mode === 'edit' ? 'Voltar ao jogo' : 'Editar poderes'}
          {mode === 'play' && pendencias.length > 0 && (
            <Badge
              variant="destructive"
              className="h-4 min-w-4 px-1 text-[10px] leading-none"
            >
              {pendencias.length}
            </Badge>
          )}
        </Button>
      </div>
      {query.trim() !== '' ? (
        <FlatAbilityResults character={character} query={query} />
      ) : mode === 'play' ? (
        <PowerPlayList character={character} />
      ) : (
        <>

      {/* Custom pill row instead of the shared Tabs primitive: this panel is
          nested inside the sheet's vertical Tabs, and Tailwind's group matching
          isn't scoped to the nearest ancestor, so a nested TabsList inherits
          the outer vertical orientation and stacks. */}
      <div className="flex shrink-0 gap-1 overflow-x-auto border-b">
        {SOURCE_TABS.map((s) => (
          <button
            key={s.value}
            type="button"
            onClick={() => setTab(s.value)}
            className={cn(
              '-mb-px flex items-center gap-1.5 whitespace-nowrap border-b-2 px-3 py-1.5 text-sm font-medium transition-colors',
              tab === s.value
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            {s.label}
            {countFor(s.value) > 0 && (
              <Badge
                variant="destructive"
                className="h-4 min-w-4 px-1 text-[10px] leading-none"
              >
                {countFor(s.value)}
              </Badge>
            )}
          </button>
        ))}
      </div>

      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {tab === 'raca' &&
          (races.length === 0 ? (
            <EmptyHint>Raça do personagem não está no catálogo.</EmptyHint>
          ) : (
            races.map((race) => (
              <RaceAbilitySection
                key={race.id}
                race={race}
                character={character}
                focus={focus}
                pending={pendingByCard.get(`raca:${race.id}`) ?? 0}
              />
            ))
          ))}

        {tab === 'origem' && (
          <OriginAbilitySection
            character={character}
            focus={focus}
            pending={pendingByCard.get('origem') ?? 0}
          />
        )}

        {tab === 'classe' &&
          (classes.length === 0 ? (
            <EmptyHint>Nenhuma classe atribuída.</EmptyHint>
          ) : (
            classes.map((entry) => (
              <ClassesSection
                key={entry.className}
                entry={entry}
                character={character}
                focus={focus}
                pending={pendingByCard.get(`classe:${entry.className}`) ?? 0}
              />
            ))
          ))}
      </div>
        </>
      )}
    </div>
  )
}

/** Flat, source-badged results for the abilities search. */
function FlatAbilityResults({
  character,
  query,
}: {
  character: Character
  query: string
}) {
  // normalize: acento-insensível ("furia" acha "Fúria") — mesmo helper das
  // outras buscas da ficha.
  const q = normalize(query.trim())
  const results = ownedAbilities(character).filter((a) =>
    normalize(a.name).includes(q),
  )
  if (results.length === 0) {
    return <EmptyHint>Nenhum poder para "{query}".</EmptyHint>
  }
  return (
    <ul className="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
      {results.map((a) => (
        <li key={`${a.source}-${a.name}`} className="rounded border border-border p-2">
          <div className="flex flex-wrap items-center gap-1.5">
            <p className="text-xs font-semibold">{a.name}</p>
            <Badge variant="secondary" className="px-1 py-0 text-[9px]">
              {a.source}
            </Badge>
            <PowerActionSlot
              spec={flatEntrySpec(a)}
              character={character}
              className="ml-auto"
            />
          </div>
          <p className="mt-0.5 text-[11px] leading-snug text-muted-foreground">
            {a.detail}
          </p>
        </li>
      ))}
    </ul>
  )
}

/**
 * Spec for a flat search result. General/Tormenta powers are outside the
 * activation registry (its id space is class/race/origin/deus) — skip them so
 * a name collision with a class power can't paint a wrong Usar button.
 */
function flatEntrySpec(a: SheetSearchEntry): ActivationSpec | undefined {
  if (a.source === 'Poder geral' || a.source === 'Poder da Tormenta') {
    return undefined
  }
  return resolveActivationSpec(a.name, a.powerId)
}

const SOURCE_TABS: { value: PendenciaSource; label: string }[] = [
  { value: 'raca', label: 'Raça' },
  { value: 'origem', label: 'Origem' },
  { value: 'classe', label: 'Classe' },
]

function EmptyHint({ children }: { children: ReactNode }) {
  return <p className={cn('text-xs italic', dimText)}>{children}</p>
}
