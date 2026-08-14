import { createMemo } from 'solid-js'
import { buffSpells } from '@/shared/lib/spell-cache'
import { PickerCombobox } from '@/shared/ui/picker-combobox'

/**
 * The GM pushes a spell buff onto a combatant. Picking fires immediately and
 * the control resets, so the same buff can be re-applied (refreshing a scene
 * buff) — buffs are never automatic, this is the explicit GM-targets-a-player
 * affordance.
 */
export function ApplyEffectSelect(props: { onApply: (spellId: string) => void }) {
  // From the primed cache: a module const would evaluate before priming.
  const options = createMemo(() => buffSpells().map((s) => ({ value: s.id, label: s.name })))
  // Um seletor COM BUSCA, não um select cru: são 31 magias, e a lista crua
  // abria uma parede que cobria a tela inteira. Mesmo controle que as condições
  // usam, pelo mesmo motivo (ALE-122).
  return (
    <PickerCombobox
      options={options()}
      onPick={props.onApply}
      aria-label="Aplicar efeito"
      placeholder="Aplicar efeito…"
      emptyMessage="Nenhuma magia encontrada."
    />
  )
}
