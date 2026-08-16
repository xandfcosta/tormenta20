import { statFor } from '@/shared/rules/items-engine'
import type { ItemEffects, WeaponStats } from '@/shared/api/item-types'
import { type AttributeKey } from '@/shared/api/attribute-keys'
import type { Character } from '@/shared/api/api'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import type { TotalContribs, WeaponCard } from '@/shared/lib/computed-sheet-v2'
import {
  areEngineCatalogsPrimed,
  computeWeaponCards as engineComputeWeaponCards,
} from '@/shared/lib/engine-wasm'
import {
  attributeTotal,
  expertiseTotalWithItems,
  parseChoiceSet,
} from './derived'
import { expertiseStateFor } from './expertise'

const ACUIDADE_POWER = 'acuidade-com-arma'

/**
 * Whether a wielded weapon may use Destreza for its attack roll / damage instead
 * of Força (T20 p145). Only helps when DES beats FOR (the rule is "pode usar", so
 * the sheet takes the better attribute). Attack finesse comes from the weapon
 * itself (Adaga: `finesse`) OR the "Acuidade com Arma" power on a light melee /
 * thrown / ágil weapon; damage finesse is Acuidade-only (the inherent weapon rule
 * is attack-only). Ranged already uses Pontaria (DES), so it never applies.
 */
function weaponDexUse(
  weapon: WeaponStats,
  hasAcuidade: boolean,
  forTotal: number,
  dexTotal: number,
): { attack: boolean; damage: boolean } {
  if (weapon.purpose === 'ranged' || dexTotal <= forTotal) {
    return { attack: false, damage: false }
  }
  const acuidade =
    hasAcuidade &&
    ((weapon.hand === 'light' && weapon.purpose === 'melee') ||
      weapon.purpose === 'thrown' ||
      weapon.traits.includes('agil'))
  return { attack: weapon.finesse === true || acuidade, damage: acuidade }
}

const EMPTY_SET: ReadonlySet<string> = new Set()

/** statFor's {total, contributions} → the ComputedSheetV2 TotalContribs shape. */
function toTotalContribs(stat: ReturnType<typeof statFor>): TotalContribs {
  return {
    total: stat.total,
    contributions: stat.contributions.map((c) => ({
      source: c.source,
      amount: c.amount,
      ...(c.note ? { note: c.note } : {}),
    })),
  }
}

/**
 * The pure TS assembly of the wielded-weapon cards from the `derived.ts` helpers
 * — the SAME payload the Go `ComputeWeaponCards` reproduces byte-equal (proven by
 * the `weaponCards` parity oracle). Kept as the single source of truth so both the
 * oracle generator and the test branch below reuse it. Skill = ranged?Pontaria:
 * Luta; damage folds Força unless ranged. Capped at two (one per hand).
 */
export function assembleWeaponCards(char: Character, effects: ItemEffects): WeaponCard[] {
  const attackAll = toTotalContribs(statFor(effects, { k: 'attack', scope: 'all' }))
  const damageAll = toTotalContribs(statFor(effects, { k: 'damage', scope: 'all' }))
  const forTotal = attributeTotal(char, 'strength', effects)
  const dexTotal = attributeTotal(char, 'dexterity', effects)
  const hasAcuidade = parseChoiceSet(char.classPowers).has(ACUIDADE_POWER)

  const cards: WeaponCard[] = []
  for (const it of char.items) {
    if (it.equipped !== 'wielded' && it.equipped !== 'wielded2') continue
    if (!it.catalogId) continue
    const weapon = getCatalogItem(it.catalogId)?.weapon
    if (!weapon) continue
    cards.push(weaponCard(char, effects, it.name, weapon, { forTotal, dexTotal, hasAcuidade, attackAll, damageAll }))
    if (cards.length === 2) break
  }
  return cards
}

/** Everything a single card needs that is the same for every weapon on the sheet. */
type CardContext = {
  forTotal: number
  dexTotal: number
  hasAcuidade: boolean
  attackAll: TotalContribs
  damageAll: TotalContribs
}

function weaponCard(
  char: Character,
  effects: ItemEffects,
  name: string,
  weapon: WeaponStats,
  context: CardContext,
): WeaponCard {
  const ranged = weapon.purpose === 'ranged'
  const skill = ranged ? 'Pontaria' : 'Luta'
  const baseAttr: AttributeKey = ranged ? 'dexterity' : 'strength'
  // Finesse (Adaga / Acuidade com Arma) swaps the attack attribute to DES when
  // it beats FOR; the perícia stays Luta but sums Destreza (ALE-31).
  const dex = weaponDexUse(weapon, context.hasAcuidade, context.forTotal, context.dexTotal)
  const attribute: AttributeKey = dex.attack ? 'dexterity' : baseAttr
  const base = expertiseStateFor(char, {
    name: skill,
    attribute: baseAttr,
    abbr: ranged ? 'DES' : 'FOR',
  })
  const state = { ...base, attribute }
  const expertise = {
    name: state.name,
    attribute: state.attribute,
    ...expertiseTotalWithItems(char, state, effects),
  }
  const strDamage = ranged ? 0 : dex.damage ? context.dexTotal : context.forTotal
  return {
    name,
    skill,
    attribute: state.attribute,
    attack: expertise.total + context.attackAll.total,
    expertise,
    attackAll: context.attackAll,
    damage: weapon.damage,
    strDamage,
    damageBonus: strDamage + context.damageAll.total,
    damageAll: context.damageAll,
    critRange: weapon.critRange,
    critMult: weapon.critMult,
  }
}

/**
 * Weapon-cards CHOKE POINT (migração TS→Go): the wielded-weapon formula cards via
 * the Go/WASM engine — the single source, em todos os ambientes. Não há mais
 * ramo TS (`assembleWeaponCards` morreu com o `t20-data`, ALE-104): o vitest
 * carrega o mesmo `.wasm` da produção.
 *
 * Takes the active conditionals as an ARGUMENT (the React version was a hook
 * reading the zustand store): the panel owns the store, this stays pure.
 *
 * @example weaponCardsFor(character, conditionals.active(character.id))
 */
export function weaponCardsFor(
  char: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): WeaponCard[] {
  if (!areEngineCatalogsPrimed()) {
    throw new Error(
      'weapon cards: WASM engine not primed — ensureEngineCatalogs() must resolve before any sheet renders',
    )
  }
  return engineComputeWeaponCards(char, [...activeConditionals])
}
