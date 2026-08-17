import { For, Show } from 'solid-js'
import type { ConditionId } from '@/shared/api/catalog-types'
import { conditionsList, conditionsRecord } from '@/shared/lib/rules-catalog-cache'
import { conditionEffectSummary } from '@/shared/rules/condition-modifiers'
import { X } from 'lucide-solid'
import { PickerCombobox } from '@/shared/ui/picker-combobox'

/**
 * Condições do livro num NPC (ALE-122, destravada pela ALE-137).
 *
 * O PC aplica condição pela FICHA, que é onde elas moram e onde o motor as lê
 * para mexer nos números. O NPC não tem ficha: as condições dele vivem na LINHA
 * da iniciativa, como estado de combate.
 *
 * E aqui elas são RASTREIO, não regra: os números do bloco de criatura são
 * escritos à mão pelo mestre, então o app não recalcula Defesa nem ataque a
 * partir da condição — dizer que recalcularia seria mentir sobre um número que
 * ninguém derivou. O chip carrega o efeito por extenso justamente por isso: é o
 * mestre quem aplica, e ele precisa ler o que a condição faz.
 *
 * @example <NpcConditions active={entry.conditions} onChange={aplicar} />
 */
export function NpcConditions(props: {
  active: string[] | undefined
  onChange: (conditions: string[]) => void
}) {
  const active = () => props.active ?? []
  const options = () =>
    conditionsList()
      .filter((condition) => !active().includes(condition.id))
      .sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
      .map((condition) => ({ value: condition.id, label: condition.name }))

  const add = (id: string) => {
    if (active().includes(id)) return
    props.onChange([...active(), id])
  }

  return (
    <section class="space-y-2" aria-label="Condições do combatente">
      <Show when={active().length > 0}>
        <ul class="flex flex-wrap gap-1.5">
          <For each={active()}>
            {(id) => (
              <NpcConditionChip
                id={id as ConditionId}
                onRemove={() => props.onChange(active().filter((c) => c !== id))}
              />
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

/**
 * Uma condição ativa com o que ela FAZ escrito ao lado — um chip que só nomeia
 * a condição é o defeito que a ALE-28 já corrigiu na ficha: no NPC ele seria
 * ainda pior, porque aqui ninguém recalcula nada e a leitura do mestre é a
 * única aplicação da regra.
 */
function NpcConditionChip(props: { id: ConditionId; onRemove: () => void }) {
  const condition = () => conditionsRecord()[props.id]

  return (
    <Show when={condition()}>
      {(found) => (
        <li
          title={found().description}
          class="flex items-center gap-1 rounded-md border border-[color:var(--hp-hurt)]/60 bg-[color:var(--hp-hurt)]/10 px-2 py-1 text-xs font-medium"
        >
          {found().name}
          <span class="text-[10px] font-normal text-muted-foreground">
            {conditionEffectSummary(props.id)}
          </span>
          <button
            type="button"
            aria-label={`Remover condição ${found().name}`}
            onClick={() => props.onRemove()}
            class="rounded p-0.5 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X aria-hidden="true" class="size-3" />
          </button>
        </li>
      )}
    </Show>
  )
}
