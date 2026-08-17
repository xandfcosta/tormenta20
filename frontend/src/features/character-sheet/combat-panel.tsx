import { createMemo } from 'solid-js'
import type { Character } from '@/shared/api/api'
import { useConditionals } from '@/shared/stores/conditionals-context'
import { AttributesGrid } from './attributes-grid'
import { CombatStats, SavesStats } from './combat-stats'
import { ContextualStatBlocks } from './contextual-stat-blocks'

/**
 * Defesa, os dois ataques, as três resistências, os seis atributos e as
 * fórmulas de arma — cada caixa abrindo a mesma decomposição de sempre.
 *
 * Estes números NÃO tinham aba nenhuma até a ALE-145: moravam só dentro do
 * `CharacterHud`, que os esconde abaixo de `md` — num telefone o jogador nunca
 * via os próprios ataques nem as resistências. Tirá-los da faixa do combatente
 * sem lhes dar casa teria apagado a informação do app inteiro nessa tela, então
 * a mudança que encolhe a faixa é a mesma que abre este bloco.
 */
export function CombatPanel(props: { character: Character }) {
  const conditionals = useConditionals()
  const active = createMemo(() => conditionals.active(props.character.id))

  return (
    <div class="min-h-0 flex-1 space-y-2 overflow-y-auto">
      <CombatStats character={props.character} activeConditionals={active()} />
      <SavesStats character={props.character} activeConditionals={active()} />
      <AttributesGrid
        character={props.character}
        activeConditionals={active()}
        class="grid-cols-6"
      />
      <ContextualStatBlocks character={props.character} activeConditionals={active()} />
    </div>
  )
}
