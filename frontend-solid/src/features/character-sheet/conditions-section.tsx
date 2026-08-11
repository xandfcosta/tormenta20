import { useQueryClient } from '@tanstack/solid-query'
import { type ConditionId, conditionEffectSummary } from '@tormenta20/t20-data'
import { X } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { conditionsList, conditionsRecord } from '@/shared/lib/rules-catalog-cache'
import { PickerCombobox } from '@/shared/ui/picker-combobox'
import { parseActiveConditions } from './active-conditions'
import { conditionActions } from './effect-mutations'

/**
 * Book conditions (caído, agarrado, atordoado… p394-395) — the #1 mid-fight
 * tracking need. Active conditions render as removable chips carrying the
 * mechanical effect they apply; the picker adds from the full catalog.
 *
 * The chips say what each condition DOES because a condition that is only a
 * badge is the bug ALE-28 was: these change the sheet's numbers.
 */
export function ConditionsSection(props: { character: Character }) {
  const queryClient = useQueryClient()
  const active = createMemo(() => parseActiveConditions(props.character.activeConditions))

  const save = async (next: ConditionId[]) => {
    try {
      await conditionActions(queryClient, props.character.id).set(next)
    } catch {
      // conditionActions already rolled back and told the player.
    }
  }

  const add = (id: string) => {
    if (active().includes(id as ConditionId)) return
    void save([...active(), id as ConditionId])
  }

  const options = createMemo(() =>
    conditionsList()
      .filter((condition) => !active().includes(condition.id))
      .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
      .map((condition) => ({ value: condition.id, label: condition.name })),
  )

  return (
    <section class="space-y-2 rounded-sm border border-grimorio-iron p-3">
      <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
        Condições (p394)
      </h3>
      <Show when={active().length > 0}>
        <ul class="flex flex-wrap gap-1.5">
          <For each={active()}>
            {(id) => (
              <ConditionChip id={id} onRemove={() => void save(active().filter((c) => c !== id))} />
            )}
          </For>
        </ul>
      </Show>
      <div class="max-w-64">
        <PickerCombobox
          options={options()}
          onPick={add}
          aria-label="Aplicar condição"
          placeholder="Aplicar condição…"
          emptyMessage="Nenhuma."
        />
      </div>
    </section>
  )
}

function ConditionChip(props: { id: ConditionId; onRemove: () => void }) {
  const condition = () => conditionsRecord()[props.id]
  return (
    <li
      title={condition().description}
      class="flex items-center gap-1 rounded-md border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/10 px-2 py-1 text-xs font-medium"
    >
      {condition().name}
      {/* The applied mechanical effect, or "lembrete" for the conditions with
          no sheet-number impact (ALE-28). */}
      <span class="text-[10px] font-normal text-muted-foreground">
        {conditionEffectSummary(props.id)}
      </span>
      <button
        type="button"
        aria-label={`Remover condição ${condition().name}`}
        onClick={() => props.onRemove()}
        class="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X aria-hidden="true" class="size-3" />
      </button>
    </li>
  )
}
