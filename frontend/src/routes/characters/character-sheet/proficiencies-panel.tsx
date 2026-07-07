import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check, X } from 'lucide-react'
import { characterProficiencies } from '@tormenta20/t20-data'
import type { ProficiencyEntry } from '@tormenta20/t20-data'
import { Button } from '@/components/ui/button'
import { api, type Character } from '@/lib/api'
import { invalidateCharacterDependents } from '@/lib/character-cache'
import { characterQueryOptions } from '@/lib/queries'
import { accentStrong, dimText, surface } from '@/lib/sheet-theme'
import { cn } from '@/lib/utils'

/**
 * "Proficiências" tab — lists every weapon / armor / shield proficiency
 * granted by the character's classes, plus a "restore class defaults"
 * button that resets manual toggles. The stored blob is a bare
 * `string[]` of category ids; anything absent from the array is
 * treated as not-granted.
 */
export function ProficienciesPanel({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const classNames = character.classes.map((c) => c.className)
  const classDefaults = characterProficiencies(classNames)
  const stored = parseProficiencySet(character.proficiencies)

  const update = useMutation<
    Character,
    Error,
    string[],
    { previous: Character | undefined }
  >({
    mutationFn: (next) =>
      api.characters.updateProficiencies(character.id, { proficiencies: next }),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, proficiencies: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: (server) => {
      qc.setQueryData<Character>(queryKey, server)
      invalidateCharacterDependents(qc, character.id)
    },
  })

  const toggle = (category: string) => {
    const next = new Set(stored)
    if (next.has(category)) next.delete(category)
    else next.add(category)
    update.mutate([...next])
  }

  const resetToDefaults = () => {
    const defaults = classDefaults.filter((e) => e.granted).map((e) => e.category)
    update.mutate(defaults)
  }

  const weapons = classDefaults.filter((e) => e.category.startsWith('armas-'))
  const armors = classDefaults.filter(
    (e) => e.category.startsWith('armaduras-') || e.category === 'escudos',
  )

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto pr-1">
      <div className="flex justify-end">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-6 px-2 text-[11px]"
          onClick={resetToDefaults}
          disabled={update.isPending}
        >
          Restaurar padrão de classe
        </Button>
      </div>
      <ProficiencyGroup
        title="Armas"
        entries={weapons}
        stored={stored}
        onToggle={toggle}
        disabled={update.isPending}
      />
      <ProficiencyGroup
        title="Armaduras & Escudos"
        entries={armors}
        stored={stored}
        onToggle={toggle}
        disabled={update.isPending}
      />
    </div>
  )
}

function parseProficiencySet(raw: string): Set<string> {
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      return new Set(parsed.filter((x): x is string => typeof x === 'string'))
    }
    return new Set()
  } catch {
    return new Set()
  }
}

function ProficiencyGroup({
  title,
  entries,
  stored,
  onToggle,
  disabled,
}: {
  title: string
  entries: ProficiencyEntry[]
  stored: Set<string>
  onToggle: (category: string) => void
  disabled: boolean
}) {
  return (
    <section className={cn('rounded-lg border p-3', surface)}>
      <h3 className={cn('font-serif text-sm font-bold', accentStrong)}>{title}</h3>
      <ul className="mt-2 space-y-1">
        {entries.map((entry) => (
          <ProficiencyRow
            key={entry.category}
            entry={entry}
            granted={stored.has(entry.category)}
            isClassDefault={entry.granted}
            onToggle={() => onToggle(entry.category)}
            disabled={disabled}
          />
        ))}
      </ul>
    </section>
  )
}

function ProficiencyRow({
  entry,
  granted,
  isClassDefault,
  onToggle,
  disabled,
}: {
  entry: ProficiencyEntry
  granted: boolean
  isClassDefault: boolean
  onToggle: () => void
  disabled: boolean
}) {
  return (
    <li
      className={cn(
        'flex items-center justify-between gap-2 rounded-md text-xs',
        granted
          ? 'bg-emerald-50 dark:bg-emerald-950/30'
          : 'bg-zinc-100 dark:bg-zinc-900/40',
      )}
    >
      <button
        type="button"
        onClick={onToggle}
        disabled={disabled}
        className="flex flex-1 items-center gap-2 rounded-md px-2 py-1 text-left hover:bg-amber-100/60 disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-zinc-800/50"
        aria-pressed={granted}
        aria-label={`${granted ? 'Remover' : 'Adicionar'} proficiência: ${entry.label}`}
      >
        {granted ? (
          <Check className="size-3.5 text-emerald-700 dark:text-emerald-400" />
        ) : (
          <X className="size-3.5 text-zinc-400" />
        )}
        <span
          className={cn(
            granted
              ? 'text-zinc-800 dark:text-zinc-100'
              : cn('line-through', dimText),
          )}
        >
          {entry.label}
        </span>
        {isClassDefault ? (
          <span
            className={cn(
              'ml-1 rounded px-1 text-[9px] uppercase tracking-wider',
              'bg-amber-200/60 text-amber-900 dark:bg-amber-500/20 dark:text-amber-200',
            )}
            title={`Padrão: ${entry.sources.join(', ')}`}
          >
            classe
          </span>
        ) : null}
      </button>
    </li>
  )
}
