import { type ReactNode, useMemo, useState } from 'react'
import { getRace } from '@tormenta20/t20-data'
import type { RaceDefinition } from '@tormenta20/t20-data'
import { Badge } from '@/shared/ui/badge'
import type { Character } from '@/shared/api/api'
import { dimText } from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import type { CardFocus } from './collapsible-ability-card'
import { ClassesSection } from './class-abilities'
import { OriginAbilitySection } from './origin-abilities'
import { RaceAbilitySection } from './race-abilities'
import {
  computePendencias,
  type Pendencia,
  type PendenciaSource,
} from './pendencias'
import { PendenciasCallout } from './pendencias-callout'

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

  const jump = (p: Pendencia) => {
    setTab(p.source)
    setFocus({ id: p.cardId, nonce: Date.now() })
  }

  const countFor = (source: PendenciaSource) =>
    pendencias.filter((p) => p.source === source).length

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
      <PendenciasCallout pendencias={pendencias} onJump={jump} />

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
    </div>
  )
}

const SOURCE_TABS: { value: PendenciaSource; label: string }[] = [
  { value: 'raca', label: 'Raça' },
  { value: 'origem', label: 'Origem' },
  { value: 'classe', label: 'Classe' },
]

function EmptyHint({ children }: { children: ReactNode }) {
  return <p className={cn('text-xs italic', dimText)}>{children}</p>
}
