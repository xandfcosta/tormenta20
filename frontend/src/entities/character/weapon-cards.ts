import { type AttributeKey, type ItemEffects, statFor } from '@tormenta20/t20-data'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import {
  areEngineCatalogsPrimed,
  computeWeaponCards as engineComputeWeaponCards,
} from '@/shared/lib/engine-wasm'
import type { TotalContribs, WeaponCard } from '@/shared/lib/computed-sheet-v2'
import type { Character } from '@/shared/api/api'
import { useActiveConditionals } from '@/shared/stores/conditionals-store'
import {
  attributeTotal,
  characterEffects,
  expertiseTotalWithItems,
} from './derived'
import { expertiseStateFor } from './expertise'

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
 * oracle generator and the hook's test branch reuse it. Skill = ranged?Pontaria:
 * Luta; damage folds Força unless ranged. Capped at two (one per hand).
 */
export function assembleWeaponCards(char: Character, effects: ItemEffects): WeaponCard[] {
  const attackAll = toTotalContribs(statFor(effects, { k: 'attack', scope: 'all' }))
  const damageAll = toTotalContribs(statFor(effects, { k: 'damage', scope: 'all' }))
  const forTotal = attributeTotal(char, 'strength', effects)

  const cards: WeaponCard[] = []
  for (const it of char.items) {
    if (it.equipped !== 'wielded' && it.equipped !== 'wielded2') continue
    if (!it.catalogId) continue
    const weapon = getCatalogItem(it.catalogId)?.weapon
    if (!weapon) continue
    const ranged = weapon.purpose === 'ranged'
    const skill = ranged ? 'Pontaria' : 'Luta'
    const attribute: AttributeKey = ranged ? 'dexterity' : 'strength'
    const state = expertiseStateFor(char, {
      name: skill,
      attribute,
      abbr: ranged ? 'DES' : 'FOR',
    })
    const expertise = {
      name: state.name,
      attribute: state.attribute,
      ...expertiseTotalWithItems(char, state, effects),
    }
    const strDamage = ranged ? 0 : forTotal
    cards.push({
      name: it.name,
      skill,
      attribute: state.attribute,
      attack: expertise.total + attackAll.total,
      expertise,
      attackAll,
      damage: weapon.damage,
      strDamage,
      damageBonus: strDamage + damageAll.total,
      damageAll,
      critRange: weapon.critRange,
      critMult: weapon.critMult,
    })
    if (cards.length === 2) break
  }
  return cards
}

/**
 * Weapon-cards CHOKE POINT (migração TS→Go): the wielded-weapon formula cards via
 * the Go/WASM engine — the single source. Same MODE-gate as `useComputedSheet` —
 * production runs the engine; the TS branch (`assembleWeaponCards`) is TEST-ONLY
 * (parity oracle, no wasm) and DCE'd from the app bundle.
 */
export function weaponCardsFor(
  char: Character,
  activeConditionals: ReadonlySet<string> = EMPTY_SET,
): WeaponCard[] {
  if (import.meta.env.MODE === 'test') {
    return assembleWeaponCards(char, characterEffects(char, activeConditionals))
  }
  if (!areEngineCatalogsPrimed()) {
    throw new Error(
      'weapon cards: WASM engine not primed — ensureEngineCatalogs() must resolve before any sheet renders',
    )
  }
  return engineComputeWeaponCards(char, [...activeConditionals])
}

/** Reactive wielded-weapon cards for a character, tracking active conditionals. */
export function useWeaponCards(char: Character): WeaponCard[] {
  const active = useActiveConditionals(char.id)
  return weaponCardsFor(char, active)
}
