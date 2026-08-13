import { startingKitFor } from '@/shared/rules/class-starting-kits'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { type RaceChoiceState, appliedRaceDeltas } from './grant-helpers'

const DEFENSE_BASE = 10
const SHIELD_LEVE_ID = 'escudo-leve'

type DefenseValues = {
  classes: { className: string; level: number }[]
  races: string[]
  dexterity: number
  startingArmor?: string
  startingShield?: boolean
}

/**
 * DEF do preview de criação, espelhando a ficha final (Go `compute.go`): base
 * 10 + Destreza final (base + delta racial) + Defesa da armadura e do escudo
 * equipados no kit inicial. Os gates `kit.armor !== 'nenhuma'` /
 * `kit.shieldLeve` copiam `startingItemsPayload` — classes sem armadura/escudo
 * no kit não contam esses bônus mesmo com os campos preenchidos. Antes o
 * preview era DEF-sem-equip (só 10 + Destreza), divergindo da ficha (ALE-26).
 *
 * **Armadura pesada zera a Destreza** (a Brunea carrega a flag
 * `cannot-apply-dex-to-defense` no catálogo, e é o motor que manda). O
 * comentário anterior aqui afirmava o oposto — "não há cap de DES" — e a Forja
 * prometia 1 de Defesa a mais do que a ficha entregava (ALE-94).
 */
export function deriveDraftDefense(
  v: DefenseValues,
  raceChoices: RaceChoiceState,
): number {
  const dexDelta = appliedRaceDeltas(v.races, raceChoices).dexterity ?? 0
  const kit = startingKitFor(v.classes[0]?.className ?? '')
  const armor =
    kit.armor !== 'nenhuma' && v.startingArmor
      ? getCatalogItem(v.startingArmor)?.armor
      : undefined
  const shieldDef =
    kit.shieldLeve && v.startingShield
      ? (getCatalogItem(SHIELD_LEVE_ID)?.shield?.defense ?? 0)
      : 0
  const dex = armor?.heavy ? 0 : v.dexterity + dexDelta
  return DEFENSE_BASE + dex + (armor?.defense ?? 0) + shieldDef
}
