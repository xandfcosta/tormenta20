import { useMemo, useState } from 'react'
import { BookOpen, Check, Search } from 'lucide-react'
import {
  SPELL_CATALOG,
  SPELL_SCHOOLS,
  SPELLCASTER_CLASSES,
  type SpellCircle,
  type SpellClassName,
  type SpellSchool,
  type SpellcasterClass,
} from '@tormenta20/t20-data'
import { Button } from '@/shared/ui/button'
import { Input } from '@/shared/ui/input'
import type { Character, CharacterSpell } from '@/shared/api/api'
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
 * Reference spellbook + persistence gate. Filters over the Cap 4
 * catalog (199 magias); rows expose Learn/Unlearn/Prepare actions via
 * SpellRow when the character owns caster classes. Persistence lives in
 * `CharacterSpell` join rows served on `character.spells`.
 */
export function SpellbookPanel({ character }: { character: Character }) {
  const casterClasses = useMemo(() => {
    return character.classes
      .map((c) => c.className)
      .filter((n): n is SpellcasterClass =>
        (SPELLCASTER_CLASSES as readonly string[]).includes(n),
      )
  }, [character.classes])

  const [query, setQuery] = useState('')
  const [circle, setCircle] = useState<SpellCircle | 'all'>('all')
  const [school, setSchool] = useState<SpellSchool | 'all'>('all')
  const [classFilter, setClassFilter] = useState<SpellClassName | 'all'>('all')
  const [learnedOnly, setLearnedOnly] = useState(false)

  const learnedById = useMemo(() => {
    const map = new Map<string, CharacterSpell>()
    for (const s of character.spells) map.set(s.catalogSpellId, s)
    return map
  }, [character.spells])

  const catalog = useMemo(() => Object.values(SPELL_CATALOG), [])

  const filtered = useMemo(() => {
    const q = query.trim() ? normalize(query) : ''
    return catalog
      .filter((s) => {
        if (circle !== 'all' && s.circle !== circle) return false
        if (school !== 'all' && s.school !== school) return false
        if (classFilter !== 'all' && !s.classes.includes(classFilter)) {
          return false
        }
        if (q && !normalize(s.name).includes(q)) return false
        if (learnedOnly && !learnedById.has(s.id)) return false
        return true
      })
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  }, [catalog, query, circle, school, classFilter, learnedOnly, learnedById])

  const noCaster = casterClasses.length === 0

  return (
    <section
      className={cn(
        'flex min-h-0 flex-col overflow-hidden rounded-xl',
        surface,
        panelBg,
      )}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-amber-700/30 px-3 py-2 dark:border-amber-500/20 sm:px-4">
        <div className="flex items-baseline gap-3">
          <h2
            className={cn(
              'flex items-center gap-2 font-serif text-lg font-bold tracking-wide',
              accentStrong,
            )}
          >
            <BookOpen className="size-4" />
            Grimório
          </h2>
          <p className={cn('text-[10px] sm:text-xs', dimText)}>
            {filtered.length} magia{filtered.length === 1 ? '' : 's'} •{' '}
            {learnedById.size} aprendida{learnedById.size === 1 ? '' : 's'}
          </p>
        </div>
        <Button
          type="button"
          variant={learnedOnly ? 'default' : 'outline'}
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={() => setLearnedOnly((v) => !v)}
        >
          <Check className="size-3.5" />
          Aprendidas
        </Button>
      </div>

      <div className="grid shrink-0 gap-2 border-b border-amber-700/20 px-3 py-2 dark:border-amber-500/15 sm:grid-cols-4 sm:px-4">
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
            className="h-7 pl-7 text-xs"
            aria-label="Buscar magia"
          />
        </div>
        <select
          value={circle === 'all' ? '' : String(circle)}
          onChange={(e) =>
            setCircle(
              e.target.value === '' ? 'all' : (Number(e.target.value) as SpellCircle),
            )
          }
          className={cn(selectClass, 'h-7 px-2 text-xs')}
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
            setSchool(e.target.value === '' ? 'all' : (e.target.value as SpellSchool))
          }
          className={cn(selectClass, 'h-7 px-2 text-xs')}
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
              e.target.value === '' ? 'all' : (e.target.value as SpellClassName),
            )
          }
          className={cn(selectClass, 'h-7 px-2 text-xs sm:col-span-2')}
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

      {noCaster && (
        <p className={cn('px-4 py-2 text-[11px]', dimText)}>
          Este personagem não tem classe conjuradora. Catálogo mostrado
          apenas como referência.
        </p>
      )}

      <div className="min-h-0 flex-1 overflow-auto px-2 py-1">
        {filtered.length === 0 ? (
          <p className={cn('px-2 py-3 text-center text-xs', dimText)}>
            Nenhuma magia para "{query}"
          </p>
        ) : (
          <div className="space-y-0.5">
            {filtered.map((spell) => (
              <SpellRow
                key={spell.id}
                spell={spell}
                character={character}
                casterClasses={casterClasses}
                learned={learnedById.get(spell.id) ?? null}
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}
