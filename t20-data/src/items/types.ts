import type { AttributeKey } from '../attributes'
import type { ExpertiseName } from '../expertises'

export type ItemCategory =
  | 'weapon-simple'
  | 'weapon-martial'
  | 'weapon-exotic'
  | 'weapon-firearm'
  | 'armor-light'
  | 'armor-heavy'
  | 'shield'
  | 'apparel'
  | 'consumable'
  | 'meal'
  | 'catalyst'
  | 'improvement'
  | 'material'
  | 'animal'
  | 'vehicle'

/**
 * Coarse grouping used to gate which catalog improvements / materials may
 * be attached to a given base item.
 *  - 'weapon'     — any weapon-* category.
 *  - 'armor'      — armor-light / armor-heavy.
 *  - 'shield'     — shields.
 *  - 'apparel'    — clothing, esotéricos, ferramentas.
 *  - 'any'        — applies to any base item.
 */
export type ItemFamily = 'weapon' | 'armor' | 'shield' | 'apparel' | 'any'

export type WeaponHand = 'light' | 'one' | 'two'

export type WeaponPurpose = 'melee' | 'thrown' | 'ranged'

export type WeaponTrait =
  | 'agil'
  | 'alongada'
  | 'desbalanceada'
  | 'dupla'
  | 'versatil'
  | 'adaptavel'

export type DamageType =
  | 'corte'
  | 'impacto'
  | 'perfuracao'
  | 'corte-perfuracao'

export type WeaponRange = 'curto' | 'medio' | 'longo'

export type WeaponStats = {
  damage: string
  critRange: number
  critMult: number
  type: DamageType
  hand: WeaponHand
  purpose: WeaponPurpose
  range?: WeaponRange
  traits: WeaponTrait[]
}

export type ArmorStats = {
  defense: number
  penalty: number
  heavy: boolean
}

export type ShieldStats = {
  defense: number
  penalty: number
  heavy: boolean
}

/**
 * Bonus categories used to enforce non-stacking rules from the rulebook.
 * Two modifiers sharing the same `bonusType` and same `target` keep only
 * the highest. `untyped` always stacks.
 *
 * Mapped to T20 conventions:
 *  - 'armor'    — armor / shield Defense bonus (don't stack with itself).
 *  - 'item'     — generic item bonus (most apparel +1 expertise).
 *  - 'training' — training bonus (level-based).
 *  - 'morale'   — temporary morale (e.g. Bardo Inspiração).
 *  - 'enhancement' — magical enhancement (melhorias).
 *  - 'untyped'  — stacks freely (situational bonuses, alchemicals).
 */
export type BonusType =
  | 'armor'
  | 'item'
  | 'training'
  | 'morale'
  | 'enhancement'
  | 'untyped'

export type EquipSlot = 'vested' | 'wielded'

export type ModifierTarget =
  | { k: 'expertise'; name: ExpertiseName }
  | { k: 'expertiseAll' }
  | { k: 'expertiseRemovePenalty'; name: ExpertiseName }
  | { k: 'expertiseByAttribute'; attribute: AttributeKey }
  | { k: 'attribute'; name: AttributeKey }
  | { k: 'defense' }
  | { k: 'defenseDexCap' }
  | { k: 'resistance' }
  | { k: 'fearResistance' }
  | { k: 'attack'; scope: 'this' | 'all' }
  | { k: 'damage'; scope: 'this' | 'all' }
  | { k: 'critRange' }
  | { k: 'critMult' }
  | { k: 'pmLimit' }
  | { k: 'pmCost' }
  /**
   * Catalisador — one-shot pmCost discount on the next magia of the
   * given school. Consumed on cast; the item consumer creates a
   * scene-scoped ActiveEffect that the cast engine looks up + deletes.
   */
  | { k: 'catalyst'; school: import('../spells').SpellSchool }
  | { k: 'spellDC' }
  | { k: 'inventorySlots' }
  | { k: 'displacement' }
  | { k: 'armorPenalty' }
  | { k: 'armorPenaltyExpertises' }
  | { k: 'tempHp' }
  | { k: 'tempMp' }
  | { k: 'maneuver'; name: 'derrubar' | 'desarmar' | 'quebrar' | 'agarrar' | 'empurrar' }
  | { k: 'flag'; name: ItemFlag }

export type ItemFlag =
  | 'lethal-unarmed'
  | 'cannot-apply-dex-to-defense'
  | 'fatigue-on-sleep'
  | 'reach-extends'

export type ModifierCondition =
  | { c: 'always' }
  | { c: 'wielded' }
  | { c: 'vested' }
  | { c: 'terrain'; type: string }
  | { c: 'against'; trait: string }
  | { c: 'context'; note: string }
  | { c: 'flagOn'; flag: string; label: string }

export type Modifier = {
  target: ModifierTarget
  amount: number
  bonusType: BonusType
  condition?: ModifierCondition
  note?: string
}

/**
 * Lifetime of an effect produced by consuming a single-use item.
 *  - 'instant'  — direct numeric mutation on consume (HP/MP). No ActiveEffect row.
 *  - 'scene'    — modifiers active until end of current scene/encounter.
 *  - 'day'      — modifiers active until next long rest. Pratos especiais use this
 *                 and are limited to 1 active per character per day.
 */
export type ConsumableScope = 'instant' | 'scene' | 'day'

/**
 * Direct numeric effect of an instant consumable (potion, healing salve, etc.).
 * Engine clients turn these into vitals patches; they don't produce modifiers.
 */
export type InstantEffect = {
  hp?: { dice: string; bonus?: number }
  mp?: { dice: string; bonus?: number }
}

export type ConsumableSpec = {
  scope: ConsumableScope
  /** True when only one instance of this item type may be active per day. */
  oncePerDay?: boolean
  /** Modifiers granted while the resulting ActiveEffect is alive. */
  modifiers?: Modifier[]
  /** Direct vitals mutation applied at consumption time. */
  instant?: InstantEffect
}

export type CatalogItem = {
  id: string
  name: string
  category: ItemCategory
  price: number
  slots: number
  /** how the item must be carried to grant its modifiers */
  equip: EquipSlot | 'either'
  /** hands occupied when wielded ('one' or 'two'); undefined when not wieldable */
  hands?: 1 | 2
  weapon?: WeaponStats
  armor?: ArmorStats
  shield?: ShieldStats
  modifiers: Modifier[]
  /** Present when the item is single-use. Consuming decrements quantity. */
  consumable?: ConsumableSpec
  /**
   * For catalog entries that overlay onto a base item — improvements
   * (melhorias) and materiais especiais. Lists which item families this
   * overlay can be attached to.
   */
  appliesTo?: ItemFamily[]
}
