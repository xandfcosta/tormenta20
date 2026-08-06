import { useMemo, useState } from 'react'
import { BookOpen, Plus, Search } from 'lucide-react'
import {
  SPELL_SCHOOLS,
  SPELLCASTER_CLASSES,
  type AttributeKey,
  type CatalogSpell,
  type SpellCircle,
  type SpellClassName,
  type SpellSchool,
  type SpellcasterClass,
} from '@tormenta20/t20-data'
import { spellCatalog } from '@/shared/lib/spell-cache'
import { Button } from '@/shared/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { VirtualList } from '@/shared/ui/virtual-list'
import type { Character, CharacterSpell } from '@/shared/api/api'
import { useComputedSheet } from '@/entities/character/computed-sheet'
import { grantedSpells } from '@/entities/character/granted-spells'
import {
  accentStrong,
  dimText,
  panelBg,
  selectClass,
  surface,
} from '@/shared/lib/sheet-theme'
import { cn } from '@/shared/lib/utils'
import { normalize } from './normalize'
import { CIRCLE_LABEL, SCHOOL_LABEL } from './spell-labels'
import { SpellRow } from './spell-row'

const CIRCLES: readonly SpellCircle[] = [0, 1, 2, 3, 4, 5]

/**
 * Character grimoire: shows only the spells the character has LEARNED, with
 * prepare/unlearn actions per row. "Aprender" opens a dialog over the full Cap
 * 4 catalog (199 magias) with círculo/escola/classe filters where new spells
 * are learned. Persistence lives in `CharacterSpell` join rows on
 * `character.spells`.
 */
export function SpellbookPanel({ character }: { character: Character }) {
  // Computed once for the whole grimoire — every SpellRow reads the same
  // per-attribute CD map the sheet's "CD Magia" box uses.
  const spellCdByAttribute = useComputedSheet(character).spellCdByAttribute
  const casterClasses = useMemo(
    () =>
      character.classes
        .map((c) => c.className)
        .filter((n): n is SpellcasterClass =>
          (SPELLCASTER_CLASSES as readonly string[]).includes(n),
        ),
    [character.classes],
  )

  const learnedById = useMemo(() => {
    const map = new Map<string, CharacterSpell>()
    for (const s of character.spells) map.set(s.catalogSpellId, s)
    return map
  }, [character.spells])

  const learned = useMemo(() => {
    const catalog = spellCatalog()
    return character.spells
      .map((s) => catalog[s.catalogSpellId])
      .filter((s): s is CatalogSpell => Boolean(s))
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  }, [character.spells],
  )

  const noCaster = casterClasses.length === 0
  // Poderes que ensinam magia (Totem Espiritual, p42) — visíveis até para
  // não-conjuradores: um Bárbaro com Totem precisa ver a magia dele.
  const granted = grantedSpells(character)

  return (
    <section
      className={cn(
        'flex min-h-0 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-border px-3 py-2 sm:px-4">
        <div className="flex items-baseline gap-3">
          <h2
            className={cn(
              'flex items-center gap-2 text-lg font-bold tracking-wide',
              accentStrong,
            )}
          >
            <BookOpen className="size-4" />
            Grimório
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            {learned.length} aprendida{learned.length === 1 ? '' : 's'}
          </p>
        </div>
        {!noCaster && (
          <LearnSpellDialog
            character={character}
            casterClasses={casterClasses}
            learnedById={learnedById}
            spellCdByAttribute={spellCdByAttribute}
          />
        )}
      </div>

      {granted.length > 0 && (
        <div className="shrink-0 space-y-1 border-b border-border px-2 py-1">
          <p className={cn('px-2 pt-1 text-[10px] font-bold uppercase tracking-widest', dimText)}>
            Concedidas por poderes
          </p>
          {granted.map((g) => (
            <SpellRow
              key={`granted-${g.spell.id}`}
              spell={g.spell}
              character={character}
              casterClasses={casterClasses}
              learned={learnedById.get(g.spell.id) ?? null}
              spellCdByAttribute={spellCdByAttribute}
              granted={{ sourcePower: g.sourcePower, keyAttribute: g.keyAttribute }}
            />
          ))}
        </div>
      )}
      {noCaster ? (
        granted.length === 0 && (
          <p className={cn('px-4 py-3 text-[11px]', dimText)}>
            Este personagem não tem classe conjuradora.
          </p>
        )
      ) : learned.length === 0 ? (
        <p className={cn('min-h-0 flex-1 px-4 py-6 text-center text-xs', dimText)}>
          Nenhuma magia aprendida. Use "Aprender" para adicionar magias.
        </p>
      ) : (
        <VirtualList
          className="min-h-0 flex-1 px-2 py-1"
          items={learned}
          estimateSize={44}
          gap={2}
          getKey={(spell) => spell.id}
          renderItem={(spell) => (
            <SpellRow
              spell={spell}
              character={character}
              casterClasses={casterClasses}
              learned={learnedById.get(spell.id) ?? null}
              spellCdByAttribute={spellCdByAttribute}
            />
          )}
        />
      )}
    </section>
  )
}

/**
 * Full-catalog learn dialog: fuzzy search + círculo/escola/classe filters over
 * the 199-spell catalog. Each row's SpellRow exposes the Learn action; already
 * learned spells show as such so the list doubles as a browse view.
 */
function LearnSpellDialog({
  character,
  casterClasses,
  learnedById,
  spellCdByAttribute,
}: {
  character: Character
  casterClasses: SpellcasterClass[]
  learnedById: Map<string, CharacterSpell>
  spellCdByAttribute: Record<AttributeKey, number>
}) {
  const [query, setQuery] = useState('')
  const [circle, setCircle] = useState<SpellCircle | 'all'>('all')
  const [school, setSchool] = useState<SpellSchool | 'all'>('all')
  const [classFilter, setClassFilter] = useState<SpellClassName | 'all'>('all')

  const catalog = useMemo(() => Object.values(spellCatalog()), [])
  const filtered = useMemo(() => {
    const q = query.trim() ? normalize(query) : ''
    return catalog
      .filter((s) => {
        if (circle !== 'all' && s.circle !== circle) return false
        if (school !== 'all' && s.school !== school) return false
        if (classFilter !== 'all' && !s.classes.includes(classFilter)) return false
        if (q && !normalize(s.name).includes(q)) return false
        return true
      })
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  }, [catalog, query, circle, school, classFilter])

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button type="button" size="sm" className="h-7 gap-1 text-xs">
          <Plus className="size-3.5" />
          Aprender
        </Button>
      </DialogTrigger>
      <DialogContent className="flex max-h-[85vh] w-[calc(100vw-1.5rem)] max-w-2xl flex-col gap-3">
        <DialogHeader>
          <DialogTitle>Aprender magia</DialogTitle>
        </DialogHeader>

        <div className="grid shrink-0 gap-2 sm:grid-cols-4">
          <div className="relative sm:col-span-2">
            <Search
              className={cn(
                'pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2',
                dimText,
              )}
            />
            <Input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Buscar magia"
              className="h-8 pl-7 text-sm"
              aria-label="Buscar magia"
            />
          </div>
          <select
            value={circle === 'all' ? '' : String(circle)}
            onChange={(e) =>
              setCircle(
                e.target.value === ''
                  ? 'all'
                  : (Number(e.target.value) as SpellCircle),
              )
            }
            className={cn(selectClass, 'h-8 px-2 text-sm')}
            aria-label="Círculo"
          >
            <option value="">Todos os círculos</option>
            {CIRCLES.map((c) => (
              <option key={c} value={c}>
                {CIRCLE_LABEL[c]}
              </option>
            ))}
          </select>
          <select
            value={school === 'all' ? '' : school}
            onChange={(e) =>
              setSchool(
                e.target.value === '' ? 'all' : (e.target.value as SpellSchool),
              )
            }
            className={cn(selectClass, 'h-8 px-2 text-sm')}
            aria-label="Escola"
          >
            <option value="">Todas as escolas</option>
            {SPELL_SCHOOLS.map((s) => (
              <option key={s} value={s}>
                {SCHOOL_LABEL[s]}
              </option>
            ))}
          </select>
          <select
            value={classFilter === 'all' ? '' : classFilter}
            onChange={(e) =>
              setClassFilter(
                e.target.value === ''
                  ? 'all'
                  : (e.target.value as SpellClassName),
              )
            }
            className={cn(selectClass, 'h-8 px-2 text-sm sm:col-span-4')}
            aria-label="Lista de classe"
          >
            <option value="">Todas as classes</option>
            {SPELLCASTER_CLASSES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </div>

        {filtered.length === 0 ? (
          <p className={cn('min-h-0 flex-1 px-2 py-6 text-center text-sm', dimText)}>
            Nenhuma magia para "{query}"
          </p>
        ) : (
          <VirtualList
            className="min-h-0 flex-1 px-1 py-1"
            items={filtered}
            estimateSize={44}
            gap={2}
            getKey={(spell) => spell.id}
            renderItem={(spell) => (
              <SpellRow
                spell={spell}
                character={character}
                casterClasses={casterClasses}
                learned={learnedById.get(spell.id) ?? null}
                spellCdByAttribute={spellCdByAttribute}
              />
            )}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
