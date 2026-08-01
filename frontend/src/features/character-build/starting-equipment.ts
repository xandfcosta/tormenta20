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

// ─── Loja (comprar com T$ iniciais, p140) ────────────────────────────

export type ShopCategoryKey =
  | 'all'
  | 'weapons'
  | 'armors'
  | 'gear'
  | 'consumables'
  | 'apparel'
  | 'animals'

export const SHOP_CATEGORIES: readonly {
  key: ShopCategoryKey
  label: string
  matches: readonly string[]
}[] = [
  { key: 'all', label: 'Todos', matches: [] },
  {
    key: 'weapons',
    label: 'Armas',
    matches: ['weapon-simple', 'weapon-martial', 'weapon-exotic', 'weapon-firearm'],
  },
  { key: 'armors', label: 'Armaduras', matches: ['armor-light', 'armor-heavy', 'shield'] },
  { key: 'gear', label: 'Equipamento', matches: ['gear', 'catalyst', 'vehicle'] },
  { key: 'consumables', label: 'Consumíveis', matches: ['consumable', 'meal'] },
  { key: 'apparel', label: 'Vestuário', matches: ['apparel'] },
  { key: 'animals', label: 'Animais', matches: ['animal'] },
]

const SHOP_EXCLUDED = new Set(['improvement', 'material', 'dr'])

/** Buyable catalog (excludes improvement/material overlays), price-sorted. */
export function shopCatalog(category: ShopCategoryKey): CatalogItem[] {
  const group = SHOP_CATEGORIES.find((c) => c.key === category)
  return CATALOG_ITEMS.filter((i) => {
    if (SHOP_EXCLUDED.has(i.category)) return false
    if (!group || group.key === 'all') return true
    return group.matches.includes(i.category)
  }).sort((a, b) => a.name.localeCompare(b.name, 'pt-BR'))
}

export type PurchaseMap = Record<string, number>

/** Total T$ spent by the purchase map (centavo-safe rounding). */
export function purchasesTotal(purchases: PurchaseMap): number {
  const cents = Object.entries(purchases).reduce((sum, [id, qty]) => {
    const item = getCatalogItem(id)
    if (!item || qty <= 0) return sum
    return sum + Math.round(item.price * 100) * qty
  }, 0)
  return cents / 100
}

/** Purchases as create-payload items (unequipped inventory rows). */
export function purchasesPayload(purchases: PurchaseMap): StartingItemPayload[] {
  return Object.entries(purchases).flatMap(([id, qty]) => {
    const item = getCatalogItem(id)
    if (!item || qty <= 0) return []
    return [{ catalogId: id, name: item.name, quantity: qty, slots: item.slots }]
  })
}
