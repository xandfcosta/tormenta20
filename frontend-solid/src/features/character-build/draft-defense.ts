import { startingKitFor } from '@tormenta20/t20-data'
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
 * equipados no kit inicial. O engine soma a Destreza cheia mesmo sob armadura
 * pesada (não há cap de DES), então o preview faz igual para bater com a ficha.
 * Os gates `kit.armor !== 'nenhuma'` / `kit.shieldLeve` copiam
 * `startingItemsPayload` — classes sem armadura/escudo no kit não contam esses
 * bônus mesmo com os campos preenchidos. Antes o preview era DEF-sem-equip (só
 * 10 + Destreza), divergindo da ficha (ALE-26).
 */
export function deriveDraftDefense(
  v: DefenseValues,
  raceChoices: RaceChoiceState,
): number {
  const dexDelta = appliedRaceDeltas(v.races, raceChoices).dexterity ?? 0
  const kit = startingKitFor(v.classes[0]?.className ?? '')
  const armorDef =
    kit.armor !== 'nenhuma' && v.startingArmor
      ? (getCatalogItem(v.startingArmor)?.armor?.defense ?? 0)
      : 0
  const shieldDef =
    kit.shieldLeve && v.startingShield
      ? (getCatalogItem(SHIELD_LEVE_ID)?.shield?.defense ?? 0)
      : 0
  return DEFENSE_BASE + v.dexterity + dexDelta + armorDef + shieldDef
}
