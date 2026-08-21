import { useQueryClient } from '@tanstack/solid-query'
import type { ProficiencyEntry } from '@/shared/api/catalog-types'
import { Check, X } from 'lucide-solid'
import { For, createMemo, createSignal } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { Button } from '@/shared/ui/button'
import { cn } from '@/shared/lib/utils'
import {
  classDefaults,
  groupProficiencies,
  ownedProficiencies,
  proficiencyActions,
  proficiencyCatalog,
  toggleProficiency,
} from './proficiency-mutations'

/**
 * The Proficiências block: every weapon / armor / shield category the
 * character's classes have an opinion about, each toggleable, plus a
 * "restaurar padrão de classe" that throws away manual edits.
 *
 * Being proficient or not is what the Mochila's "sem proficiência" warning
 * reads — this is where that warning gets resolved.
 */
export function ProficienciesPanel(props: { character: Character }) {
  const queryClient = useQueryClient()
  const actions = () => proficiencyActions(queryClient, props.character.id)
  const [pending, setPending] = createSignal(false)

  const owned = createMemo(() => ownedProficiencies(props.character))
  const groups = createMemo(() => groupProficiencies(proficiencyCatalog(props.character)))

  const save = async (categories: string[]) => {
    setPending(true)
    try {
      await actions().set(categories)
    } catch {
      // proficiencyActions already rolled back and told the player.
    } finally {
      setPending(false)
    }
  }

  return (
    <section class="flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-none border border-grimorio-iron bg-grimorio-panel">
      <div class="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-grimorio-iron px-3 py-2 sm:px-4">
        <h2 class="font-heading text-lg uppercase tracking-wide text-grimorio-gold">
          Proficiências
        </h2>
        <Button
          type="button"
          variant="outline"
          size="sm"
          class="h-7 text-xs"
          disabled={pending()}
          onClick={() => void save(classDefaults(props.character))}
        >
          Restaurar padrão de classe
        </Button>
      </div>

      <div class="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-3 sm:p-4">
        <ProficiencyGroup
          title="Armas"
          entries={groups().weapons}
          owned={owned()}
          disabled={pending()}
          onToggle={(category) => void save(toggleProficiency(owned(), category))}
        />
        <ProficiencyGroup
          title="Armaduras & Escudos"
          entries={groups().armors}
          owned={owned()}
          disabled={pending()}
          onToggle={(category) => void save(toggleProficiency(owned(), category))}
        />
      </div>
    </section>
  )
}

function ProficiencyGroup(props: {
  title: string
  entries: ProficiencyEntry[]
  owned: ReadonlySet<string>
  disabled: boolean
  onToggle: (category: string) => void
}) {
  return (
    <section class="rounded-none border border-grimorio-iron p-3">
      <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
        {props.title}
      </h3>
      <ul class="mt-2 space-y-1">
        <For each={props.entries}>
          {(entry) => (
            <ProficiencyRow
              entry={entry}
              granted={props.owned.has(entry.category)}
              disabled={props.disabled}
              onToggle={() => props.onToggle(entry.category)}
            />
          )}
        </For>
      </ul>
    </section>
  )
}

function ProficiencyRow(props: {
  entry: ProficiencyEntry
  granted: boolean
  disabled: boolean
  onToggle: () => void
}) {
  return (
    <li class={cn('rounded-sm text-xs', props.granted && 'bg-emerald-950/30')}>
      <button
        type="button"
        onClick={() => props.onToggle()}
        disabled={props.disabled}
        aria-pressed={props.granted}
        aria-label={`${props.granted ? 'Remover' : 'Adicionar'} proficiência: ${props.entry.label}`}
        class="flex w-full items-center gap-2 rounded-sm px-2 py-1 text-left transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-60"
      >
        {props.granted ? (
          <Check aria-hidden="true" class="size-3.5 text-emerald-400" />
        ) : (
          <X aria-hidden="true" class="size-3.5 text-muted-foreground" />
        )}
        <span
          class={cn(
            props.granted ? 'text-foreground' : 'text-muted-foreground line-through',
          )}
        >
          {props.entry.label}
        </span>
        {/* "classe" marks what the class grants by default, so a player can
            tell an intentional manual toggle from the baseline. */}
        {props.entry.granted && (
          <span
            class="ml-1 rounded-md bg-muted px-1 text-[9px] uppercase tracking-wider text-muted-foreground"
            title={`Padrão: ${props.entry.sources.join(', ')}`}
          >
            classe
          </span>
        )}
      </button>
    </li>
  )
}
