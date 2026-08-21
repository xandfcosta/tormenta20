import { useQueryClient } from '@tanstack/solid-query'
import { conditionEffectSummary } from '@/shared/rules/condition-modifiers'
import type { ConditionId } from '@/shared/api/catalog-types'
import { X } from 'lucide-solid'
import { For, Show, createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { conditionsList, conditionsRecord } from '@/shared/lib/rules-catalog-cache'
import { cn } from '@/shared/lib/utils'
import { PickerCombobox } from '@/shared/ui/picker-combobox'
import { parseActiveConditions } from './active-conditions'
import { conditionActions } from './effect-mutations'

export type ConditionEditing = {
  active: () => ConditionId[]
  /** As condições que ainda cabem, já ordenadas em pt-BR, para o picker. */
  options: () => { value: string; label: string }[]
  add: (id: string) => void
  remove: (id: ConditionId) => void
}

/**
 * Aplicar e remover condições do livro (p394-395). Vive fora do componente para
 * que a aba Efeitos e a faixa do combatente (ALE-145) editem a MESMA coisa sem
 * uma copiar a lógica da outra — o que elas não compartilham é a forma.
 *
 * @example const conditions = createConditionEditing(() => props.character)
 */
export function createConditionEditing(character: () => Character): ConditionEditing {
  const queryClient = useQueryClient()
  const active = createMemo(() => parseActiveConditions(character().activeConditions))

  const save = async (next: ConditionId[]) => {
    try {
      await conditionActions(queryClient, character().id).set(next)
    } catch {
      // conditionActions already rolled back and told the player.
    }
  }

  const options = createMemo(() =>
    conditionsList()
      .filter((condition) => !active().includes(condition.id))
      .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
      .map((condition) => ({ value: condition.id, label: condition.name })),
  )

  return {
    active,
    options,
    add: (id) => {
      if (active().includes(id as ConditionId)) return
      void save([...active(), id as ConditionId])
    },
    remove: (id) => void save(active().filter((c) => c !== id)),
  }
}

/**
 * Book conditions (caído, agarrado, atordoado… p394-395) — the #1 mid-fight
 * tracking need. Active conditions render as removable chips carrying the
 * mechanical effect they apply; the picker adds from the full catalog.
 *
 * The chips say what each condition DOES because a condition that is only a
 * badge is the bug ALE-28 was: these change the sheet's numbers.
 */
export function ConditionsSection(props: { character: Character }) {
  const conditions = createConditionEditing(() => props.character)

  return (
    <section class="space-y-2 rounded-sm border border-grimorio-iron p-3">
      <h3 class="font-heading text-sm uppercase tracking-wide text-grimorio-gold">
        Condições (p394)
      </h3>
      <Show when={conditions.active().length > 0}>
        <ul class="flex flex-wrap gap-1.5">
          <For each={conditions.active()}>
            {(id) => <ConditionChip id={id} onRemove={() => conditions.remove(id)} />}
          </For>
        </ul>
      </Show>
      <div class="max-w-64">
        <PickerCombobox
          options={conditions.options()}
          onPick={conditions.add}
          aria-label="Aplicar condição"
          placeholder="Aplicar condição…"
          emptyMessage="Nenhuma."
        />
      </div>
    </section>
  )
}

/**
 * Uma condição ativa, com o botão de tirar. `compact` esconde o resumo do
 * efeito: na faixa do combatente o que se lê é "quem está caído", e o resumo
 * ("−2 em testes de FOR…") dobrava a largura do chip.
 */
export function ConditionChip(props: {
  id: ConditionId
  onRemove: () => void
  compact?: boolean
}) {
  const condition = () => conditionsRecord()[props.id]
  return (
    <li
      title={condition().description}
      class={cn(
        'flex items-center gap-1 rounded-md border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/10 font-medium',
        props.compact ? 'px-1.5 py-px text-[11px]' : 'px-2 py-1 text-xs',
      )}
    >
      {condition().name}
      {/* The applied mechanical effect, or "lembrete" for the conditions with
          no sheet-number impact (ALE-28). */}
      <Show when={!props.compact}>
        <span class="text-[10px] font-normal text-muted-foreground">
          {conditionEffectSummary(props.id)}
        </span>
      </Show>
      <button
        type="button"
        aria-label={`Remover condição ${condition().name}`}
        onClick={() => props.onRemove()}
        class="rounded-lg p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
      >
        <X aria-hidden="true" class="size-3" />
      </button>
    </li>
  )
}
