import { Show } from 'solid-js'
import type { StatsProps } from './combat-stats'
import { isCasterCharacter } from './is-caster'
import { MagicStats } from './magic-stats'
import { WeaponFormulaCards } from './weapon-formula-cards'
import { hasWieldedWeapon } from './wielded-weapons'

/**
 * The two blocks that only matter sometimes: weapon formulas whenever
 * something is wielded — a hybrid (Paladino, Bardo) keeps its attack row — and
 * the magic triple for any caster. A martial with empty hands still gets the
 * "Nenhuma arma empunhada" placeholder, so the slot never looks broken.
 */
export function ContextualStatBlocks(props: StatsProps) {
  const wielding = () => hasWieldedWeapon(props.character)
  const caster = () => isCasterCharacter(props.character)

  return (
    <>
      <Show when={wielding() || !caster()}>
        <WeaponFormulaCards
          character={props.character}
          activeConditionals={props.activeConditionals}
        />
      </Show>
      <Show when={caster()}>
        <MagicStats character={props.character} activeConditionals={props.activeConditionals} />
      </Show>
    </>
  )
}
