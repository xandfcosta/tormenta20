import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { BookPlus, BookX, Check, Sparkles } from 'lucide-react'
import {
  CLASS_SPELLCASTING_ATTRIBUTE,
  SPELL_BASE_PM_COST,
  highestCircleAtLevel,
  spellSaveDc,
  type CatalogSpell,
  type SpellCircle,
  type SpellcasterClass,
} from '@tormenta20/t20-data'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import type { AttributeKey, Character, CharacterSpell } from '@/lib/api'
import { api } from '@/lib/api'
import { invalidateCharacterDependents } from '@/lib/character-cache'
import { characterQueryOptions } from '@/lib/queries'
import { accentStrong, dimText, hoverRow } from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'
import { CIRCLE_LABEL, SCHOOL_LABEL } from './spell-labels'

const ATTRIBUTE_MAP: Record<AttributeKey, keyof Character> = {
  strength: 'strength',
  dexterity: 'dexterity',
  constitution: 'constitution',
  intelligence: 'intelligence',
  wisdom: 'wisdom',
  charisma: 'charisma',
}

/**
 * One catalog row. Header collapses to name + círculo + escola + PM +
 * CD + status badge; expands to full stats + augment list + action
 * cluster (Aprender / Preparar / Esquecer). Mutations invalidate the
 * character query so persisted spells reflect immediately in the
 * "Aprendidas" filter.
 */
export function SpellRow({
  spell,
  character,
  casterClasses,
  learned,
}: {
  spell: CatalogSpell
  character: Character
  casterClasses: readonly SpellcasterClass[]
  learned: CharacterSpell | null
}) {
  const [open, setOpen] = useState(false)
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey

  const applicableClasses = casterClasses.filter((c) =>
    spell.classes.includes(c),
  )
  const bestCd = computeBestCd(character, applicableClasses)
  const cast = highestCastableCircle(character, applicableClasses)
  const canCast = spell.circle <= cast

  const learn = useMutation({
    mutationFn: () => api.characters.learnSpell(character.id, spell.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey })
      invalidateCharacterDependents(qc, character.id)
    },
  })
  const unlearn = useMutation({
    mutationFn: () => api.characters.unlearnSpell(character.id, spell.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey })
      invalidateCharacterDependents(qc, character.id)
    },
  })
  const setPrepared = useMutation({
    mutationFn: (prepared: boolean) =>
      api.characters.setSpellPrepared(character.id, spell.id, prepared),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey })
    },
  })

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
        {learned && (
          <Badge
            variant={learned.prepared ? 'default' : 'secondary'}
            className="text-[10px] uppercase tracking-widest"
          >
            {learned.prepared ? 'Preparada' : 'Aprendida'}
          </Badge>
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
          <div className="flex flex-wrap items-center gap-2 border-t border-amber-700/10 pt-2 dark:border-amber-500/10">
            {learned ? (
              <>
                <Button
                  type="button"
                  size="sm"
                  variant={learned.prepared ? 'default' : 'outline'}
                  className="h-7 gap-1 text-xs"
                  disabled={setPrepared.isPending}
                  onClick={() => setPrepared.mutate(!learned.prepared)}
                >
                  <Check className="size-3.5" />
                  {learned.prepared ? 'Despreparar' : 'Preparar'}
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="h-7 gap-1 text-xs text-red-700 hover:bg-red-100 dark:text-red-400 dark:hover:bg-red-950/40"
                  disabled={unlearn.isPending}
                  onClick={() => unlearn.mutate()}
                >
                  <BookX className="size-3.5" />
                  Esquecer
                </Button>
              </>
            ) : (
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="h-7 gap-1 text-xs"
                disabled={learn.isPending}
                onClick={() => learn.mutate()}
              >
                <BookPlus className="size-3.5" />
                Aprender
              </Button>
            )}
          </div>
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
