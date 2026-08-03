import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import { CONDITIONS, type ConditionId } from '@tormenta20/t20-data'
import { api, type Character } from '@/shared/api/api'
import { Combobox } from '@/shared/ui/combobox'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/shared/ui/popover'
import { cn } from '@/shared/lib/utils'
import { invalidateCharacterDependents } from '@/entities/character/character-cache'
import { characterQueryOptions } from '@/entities/character/queries'

/** Parse the persisted ConditionId[] blob (bad blob ⇒ none). */
export function parseActiveConditions(raw: string): ConditionId[] {
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is ConditionId => typeof x === 'string' && x in CONDITIONS)
  } catch {
    return []
  }
}

/**
 * Book conditions (caído, agarrado, atordoado… PDF p394-395) — the #1
 * mid-fight tracking need. Active conditions render as removable chips with
 * the rule text on hover; the picker adds from the full t20-data catalog.
 * Optimistic: the chip appears/disappears instantly, rolls back on failure.
 */
export function ConditionsSection({ character }: { character: Character }) {
  const qc = useQueryClient()
  const queryKey = characterQueryOptions(character.id).queryKey
  const active = parseActiveConditions(character.activeConditions)

  const update = useMutation({
    mutationFn: (next: ConditionId[]) =>
      api.characters.updateConditions(character.id, next),
    onMutate: async (next) => {
      await qc.cancelQueries({ queryKey })
      const previous = qc.getQueryData<Character>(queryKey)
      qc.setQueryData<Character>(queryKey, (prev) =>
        prev ? { ...prev, activeConditions: JSON.stringify(next) } : prev,
      )
      return { previous }
    },
    onError: (_e, _v, ctx) => {
      if (ctx?.previous) qc.setQueryData(queryKey, ctx.previous)
    },
    onSuccess: () => invalidateCharacterDependents(qc, character.id),
  })

  const add = (id: string) => {
    if (!id || active.includes(id as ConditionId)) return
    update.mutate([...active, id as ConditionId])
  }
  const remove = (id: ConditionId) =>
    update.mutate(active.filter((c) => c !== id))

  const options = Object.values(CONDITIONS)
    .filter((c) => !active.includes(c.id))
    .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
    .map((c) => ({ value: c.id, label: c.name }))

  return (
    // scroll-mt clears the sticky TopNav (~53px) when this section is the
    // scroll target, so the header never lands hidden under it (audit task 11).
    <div className="scroll-mt-14 space-y-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">
        Condições (p394)
      </p>
      {active.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {active.map((id) => {
            const cond = CONDITIONS[id]
            return (
              <li
                key={id}
                title={cond.description}
                className={cn(
                  'flex items-center gap-1 rounded-md border border-[color:var(--hp-hurt)]/60',
                  'bg-[color:var(--hp-hurt)]/10 px-2 py-1 text-xs font-medium',
                )}
              >
                {cond.name}
                <button
                  type="button"
                  aria-label={`Remover condição ${cond.name}`}
                  onClick={() => remove(id)}
                  className="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
                >
                  <X className="size-3" />
                </button>
              </li>
            )
          })}
        </ul>
      )}
      <div className="max-w-64">
        <Combobox
          options={options}
          value=""
          onChange={add}
          placeholder="Aplicar condição…"
          searchPlaceholder="Buscar condição…"
          emptyMessage="Nenhuma."
        />
      </div>
    </div>
  )
}

/** Compact HUD pips: active conditions at a glance on every viewport. */
export function ConditionPips({
  character,
  className,
  mini = false,
}: {
  character: Character
  className?: string
  /** Compact HUD variant: h-4 chips inline with the class badges, 4+ folds
   *  into a ⚠+N popover — a dedicated row doubled the nameplate height. */
  mini?: boolean
}) {
  const active = parseActiveConditions(character.activeConditions)
  if (active.length === 0) return null
  const shown = mini ? active.slice(0, 3) : active
  const overflow = active.length - shown.length
  const chip = mini
    ? 'max-w-20 truncate rounded border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/15 px-1 text-[9px] font-semibold uppercase tracking-wide text-[color:var(--hp-hurt)]'
    : 'rounded border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/15 px-1.5 py-px text-[10px] font-semibold uppercase tracking-wide text-[color:var(--hp-hurt)]'
  return (
    <ul className={cn('flex flex-wrap items-center gap-1', className)}>
      {shown.map((id) => (
        <li key={id} title={CONDITIONS[id].description} className={chip}>
          {CONDITIONS[id].name}
        </li>
      ))}
      {overflow > 0 && (
        <li>
          <Popover>
            <PopoverTrigger asChild>
              <button type="button" className={chip} aria-label={`Mais ${overflow} condições`}>
                ⚠+{overflow}
              </button>
            </PopoverTrigger>
            <PopoverContent className="w-72 space-y-2 text-xs">
              {active.map((id) => (
                <div key={id}>
                  <p className="font-semibold uppercase">{CONDITIONS[id].name}</p>
                  <p className="text-muted-foreground">
                    {CONDITIONS[id].description}
                  </p>
                </div>
              ))}
            </PopoverContent>
          </Popover>
        </li>
      )}
    </ul>
  )
}
