import { useMemo, useState } from 'react'
import { BookOpen, Search, Sparkles } from 'lucide-react'
import {
  CLASS_SPELLCASTING_ATTRIBUTE,
  SPELL_BASE_PM_COST,
  SPELL_CATALOG,
  SPELL_SCHOOLS,
  SPELLCASTER_CLASSES,
  highestCircleAtLevel,
  spellSaveDc,
  type CatalogSpell,
  type SpellCircle,
  type SpellClassName,
  type SpellSchool,
  type SpellcasterClass,
} from '@tormenta20/t20-data'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import type { AttributeKey, Character } from '@/lib/api'
import {
  accentStrong,
  dimText,
  hoverRow,
  panelBg,
  selectClass,
  surface,
} from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
import { normalize } from './normalize'

const SCHOOL_LABEL: Record<SpellSchool, string> = {
  abjuracao: 'Abjuração',
  adivinhacao: 'Adivinhação',
  convocacao: 'Convocação',
  encantamento: 'Encantamento',
  evocacao: 'Evocação',
  ilusao: 'Ilusão',
  necromancia: 'Necromancia',
  transmutacao: 'Transmutação',
}

const CIRCLES: readonly SpellCircle[] = [0, 1, 2, 3, 4, 5]

const CIRCLE_LABEL: Record<SpellCircle, string> = {
  0: 'Truque',
  1: '1º',
  2: '2º',
  3: '3º',
  4: '4º',
  5: '5º',
}

const ATTRIBUTE_MAP: Record<AttributeKey, keyof Character> = {
  strength: 'strength',
  dexterity: 'dexterity',
  constitution: 'constitution',
  intelligence: 'intelligence',
  wisdom: 'wisdom',
  charisma: 'charisma',
}

/**
 * Reference spellbook — read-only browser over the Cap 4 catalog
 * (199 magias). Filters by class list (auto-scoped to the character's
 * caster classes when they have any), circle, school, and free-text
 * search. Save CD is computed per-spell using the character's caster
 * class + level + key attribute per PDF p171 formula.
 *
 * Persistence (learn / prepare / cast) intentionally out of scope for
 * this PR — this is the consumer integration bit called out in
 * ROADMAP; the engine ships in follow-up PRs.
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
        return true
      })
      .sort((a, b) => a.circle - b.circle || a.name.localeCompare(b.name))
  }, [catalog, query, circle, school, classFilter])

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
            {filtered.length} magia{filtered.length === 1 ? '' : 's'} •
            catálogo Cap 4 (referência)
          </p>
        </div>
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
              />
            ))}
          </div>
        )}
      </div>
    </section>
  )
}

/**
 * One catalog row. Shows a compact header (name + circle + school +
 * PM cost + save CD) and expands to full stats + augments on click.
 * CD is computed per applicable caster class the character has —
 * `—` when no caster class overlaps with the spell's class list.
 */
function SpellRow({
  spell,
  character,
  casterClasses,
}: {
  spell: CatalogSpell
  character: Character
  casterClasses: readonly SpellcasterClass[]
}) {
  const [open, setOpen] = useState(false)

  const applicableClasses = casterClasses.filter((c) =>
    spell.classes.includes(c),
  )
  const bestCd = computeBestCd(spell, character, applicableClasses)
  const cast = highestCastableCircle(character, applicableClasses)
  const canCast = spell.circle <= cast

  return (
    <div
      className={cn(
        'rounded-md border border-transparent px-2 py-1',
        hoverRow,
        open && 'border-amber-700/20 dark:border-amber-500/15',
      )}
    >
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center gap-2 text-left"
        aria-expanded={open}
      >
        <Badge
          variant="outline"
          className={cn('font-mono text-[10px]', !canCast && dimText)}
        >
          {CIRCLE_LABEL[spell.circle]}
        </Badge>
        <span className="flex-1 truncate text-sm font-medium">
          {spell.name}
        </span>
        <span className={cn('text-[10px] uppercase tracking-widest', dimText)}>
          {SCHOOL_LABEL[spell.school]}
        </span>
        <span className={cn('font-mono text-xs', accentStrong)}>
          {SPELL_BASE_PM_COST[spell.circle]} PM
        </span>
        {bestCd !== null && (
          <span className="font-mono text-xs text-violet-700 dark:text-violet-300">
            CD {bestCd}
          </span>
        )}
      </button>
      {open && (
        <div className="mt-2 space-y-2 border-t border-amber-700/15 pt-2 text-xs dark:border-amber-500/10">
          <div className="grid gap-1 sm:grid-cols-2">
            <Stat label="Execução" value={spell.execution} />
            <Stat label="Alcance" value={spell.range} />
            <Stat
              label="Duração"
              value={
                spell.duration === 'definida' && spell.durationNote
                  ? spell.durationNote
                  : spell.duration
              }
            />
            <Stat label="Resistência" value={spell.resistance ?? '—'} />
            <Stat label="Teste" value={spell.saveType} />
            <Stat label="Livro" value={`p${spell.bookPage}`} />
          </div>
          <p className="whitespace-pre-line text-zinc-700 dark:text-zinc-300">
            {spell.baseEffect}
          </p>
          {spell.augments.length > 0 && (
            <div className="space-y-1">
              <p
                className={cn(
                  'text-[10px] uppercase tracking-widest',
                  dimText,
                )}
              >
                Aprimoramentos
              </p>
              <ul className="space-y-1">
                {spell.augments.map((a, i) => (
                  <li
                    key={i}
                    className="flex items-start gap-2 rounded border border-amber-700/10 px-2 py-1 dark:border-amber-500/10"
                  >
                    <Badge
                      variant="outline"
                      className="font-mono text-[10px]"
                    >
                      +{a.pmCost} PM
                    </Badge>
                    <span
                      className={cn(
                        'text-[10px] uppercase tracking-widest',
                        a.kind === 'muda'
                          ? 'text-violet-700 dark:text-violet-300'
                          : 'text-emerald-700 dark:text-emerald-300',
                      )}
                    >
                      {a.kind}
                    </span>
                    <span className="flex-1 text-zinc-700 dark:text-zinc-300">
                      {a.description}
                      {a.classOnly && (
                        <span className={cn('ml-1 italic', dimText)}>
                          (apenas {a.classOnly})
                        </span>
                      )}
                      {a.requiresCircle !== undefined && (
                        <span className={cn('ml-1 italic', dimText)}>
                          (requer {CIRCLE_LABEL[a.requiresCircle]})
                        </span>
                      )}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {!canCast && applicableClasses.length > 0 && (
            <p className="flex items-center gap-1 text-[11px] text-amber-700 dark:text-amber-300">
              <Sparkles className="size-3" />
              Círculo acima do máximo conjurável no nível atual.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-baseline gap-1">
      <span
        className={cn('text-[10px] uppercase tracking-widest', dimText)}
      >
        {label}
      </span>
      <span className="text-zinc-800 dark:text-zinc-200">{value}</span>
    </div>
  )
}

function computeBestCd(
  _spell: CatalogSpell,
  character: Character,
  applicableClasses: readonly SpellcasterClass[],
): number | null {
  if (applicableClasses.length === 0) return null
  let best = -Infinity
  for (const c of applicableClasses) {
    const attr = CLASS_SPELLCASTING_ATTRIBUTE[c]
    if (!attr) continue
    const attrValue = character[ATTRIBUTE_MAP[attr]] as number
    const dc = spellSaveDc(character.level, attrValue)
    if (dc > best) best = dc
  }
  return best === -Infinity ? null : best
}

function highestCastableCircle(
  character: Character,
  applicableClasses: readonly SpellcasterClass[],
): SpellCircle {
  let best: SpellCircle = 0
  for (const c of applicableClasses) {
    const classEntry = character.classes.find((cc) => cc.className === c)
    if (!classEntry) continue
    const circle = highestCircleAtLevel(c, classEntry.level)
    if (circle > best) best = circle
  }
  return best
}
