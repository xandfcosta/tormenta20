import {
  CATALOG_ITEMS,
  type CatalogItem,
  getCatalogItem,
  ORIGENS,
  type StartingKit,
  startingKitFor,
  startingMoneyForLevel,
} from '@tormenta20/t20-data'

/**
 * Equipamento inicial (book p140) helpers for the wizard step: the unified L1
 * kit narrowed by class proficiências (`startingKitFor`), weapon/armor picker
 * pools from the item catalog, origem itens, and the Tabela 3-1 money default.
 * Homebrew: kit pickers stay available at any level; money defaults to the
 * level's table value (editable).
 */

export const LIGHT_ARMOR_IDS = [
  'armadura-couro',
  'couro-batido',
  'gibao-peles',
] as const

export const KIT_BASE_ITEM_IDS = [
  'mochila',
  'saco-de-dormir',
  'traje-viajante',
] as const

export type StartingEquipmentDraft = {
  weaponSimple: string
  weaponMartial: string
  armor: string
  shield: boolean
}

export function weaponOptions(
  category: 'weapon-simple' | 'weapon-martial',
): CatalogItem[] {
  return CATALOG_ITEMS.filter((i) => i.category === category)
}

export function lightArmorOptions(): CatalogItem[] {
  return LIGHT_ARMOR_IDS.map((id) => getCatalogItem(id)).filter(
    (i): i is CatalogItem => i !== undefined,
  )
}

/** Kit for the class + Tabela 3-1 default money for the level (1 → null: 4d6). */
export function startingLoadout(
  className: string,
  level: number,
): { kit: StartingKit; tableMoney: number | null } {
  const clamped = Math.min(20, Math.max(1, level || 1))
  return {
    kit: startingKitFor(className),
    tableMoney: startingMoneyForLevel(clamped),
  }
}

/** Origem itens iniciais (auto-granted) — display names, by origem NAME. */
export function originStartingItems(originName: string): readonly string[] {
  const origem = Object.values(ORIGENS).find((o) => o.name === originName)
  return origem?.itensIniciais ?? []
}

export type StartingItemPayload = {
  catalogId?: string
  name: string
  quantity?: number
  slots?: number
  equipped?: string
}

function catalogPayload(id: string, equipped?: string): StartingItemPayload | null {
  const item = getCatalogItem(id)
  if (!item) return null
  return {
    catalogId: id,
    name: item.name,
    quantity: 1,
    slots: item.slots,
    equipped,
  }
}

/**
 * The create-payload items for the chosen kit + origem grants: base items,
 * picked weapons (wielded), armor (vested), escudo leve (wielded), origem
 * itens as catalog-less rows. Empty picks are simply skipped — under-filling
 * is allowed (finish on the sheet).
 */
export function startingItemsPayload(
  draft: StartingEquipmentDraft,
  kit: StartingKit,
  originName: string,
): StartingItemPayload[] {
  const out: StartingItemPayload[] = []
  for (const id of KIT_BASE_ITEM_IDS) {
    const item = catalogPayload(id)
    if (item) out.push(item)
  }
  if (draft.weaponSimple) {
    const w = catalogPayload(draft.weaponSimple, 'wielded')
    if (w) out.push(w)
  }
  if (kit.weapons === 'simples+marcial' && draft.weaponMartial) {
    const w = catalogPayload(draft.weaponMartial, 'wielded')
    if (w) out.push(w)
  }
  if (kit.armor !== 'nenhuma' && draft.armor) {
    const a = catalogPayload(draft.armor, 'vested')
    if (a) out.push(a)
  }
  if (kit.shieldLeve && draft.shield) {
    const s = catalogPayload('escudo-leve', 'wielded')
    if (s) out.push(s)
  }
  for (const name of originStartingItems(originName)) {
    out.push({ name, quantity: 1, slots: 1 })
  }
  return out
}
