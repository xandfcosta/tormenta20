import { getCatalogItem } from '@/shared/lib/catalog-cache'
import type { CatalogItem, ItemFlag } from '@/shared/api/item-types'
import type { Modifier } from '@/shared/api/item-types'
import { signed } from './signed'

/**
 * PT-BR labels for the boolean item flags (`{ k: 'flag' }` targets). Flags
 * are on/off — renderers must never append an amount to these labels.
 * Kept beside `describeModifierTarget` so the slug→label mapping exists
 * exactly once in the frontend.
 */
const ITEM_FLAG_LABELS: Record<ItemFlag, string> = {
  'lethal-unarmed': 'Ataques desarmados causam dano letal',
  'cannot-apply-dex-to-defense': 'Não soma Destreza na Defesa',
  'fatigue-on-sleep': 'Fadiga ao dormir',
  'reach-extends': 'Alcance ampliado',
  'armadura-pesada': 'Conta como armadura pesada',
  'auto-fail-reflexos': 'Falha automática em Reflexos',
}

/**
 * Human-readable labels for the two Modifier fields that need to be
 * displayed inside the item info + catalog dialogs. Kept exhaustive
 * so a new Modifier variant surfaces as a TS error rather than a
 * silently-empty catalog row.
 */
export function describeModifierTarget(t: Modifier['target']): string {
  switch (t.k) {
    case 'expertise':
      return `Perícia ${t.name}`
    case 'expertiseAll':
      return 'Todas perícias'
    case 'expertiseRemovePenalty':
      return `Remove penalidade em ${t.name}`
    case 'expertiseByAttribute':
      return `Perícias de ${t.attribute}`
    case 'attribute':
      return `Atributo ${t.name}`
    case 'defense':
      return 'Defesa'
    case 'defenseDexCap':
      return 'Limite de Des na Defesa'
    case 'resistance':
      return 'Resistências'
    case 'fearResistance':
      return 'Resistência a medo'
    case 'attack':
      return `Ataque (${t.scope})`
    case 'damage':
      return `Dano (${t.scope})`
    case 'critRange':
      return 'Margem de ameaça'
    case 'critMult':
      return 'Multiplicador crítico'
    case 'pmLimit':
      return 'Limite de PM por magia'
    case 'pmCost':
      return 'Custo em PM'
    case 'damageReduction':
      return 'Redução de dano'
    case 'catalyst':
      return `Catalisador (${t.school})`
    case 'spellDC':
      return 'CD de magias'
    case 'inventorySlots':
      return 'Espaços de carga'
    case 'displacement':
      return 'Deslocamento'
    case 'flySpeed':
      return 'Voo'
    case 'armorPenalty':
      return 'Penalidade de armadura'
    case 'armorPenaltyExpertises':
      return 'Penalidade em perícias'
    case 'tempHp':
      return 'PV temporários'
    case 'tempMp':
      return 'PM temporários'
    case 'maxPv':
      return 'PV máximo'
    case 'maxPm':
      return 'PM máximo'
    case 'maneuver':
      return `Manobra ${t.name}`
    case 'flag':
      return ITEM_FLAG_LABELS[t.name]
  }
}

export function describeCondition(m: Modifier): string | null {
  if (!m.condition) return null
  switch (m.condition.c) {
    case 'always':
      return null
    case 'wielded':
      return 'enquanto empunhado'
    case 'vested':
      return 'enquanto vestido'
    case 'flagOff':
      return m.condition.label
    case 'terrain':
      return `terreno: ${m.condition.type}`
    case 'against':
      return `contra: ${m.condition.trait}`
    case 'context':
      return m.condition.note
    case 'flagOn':
      return m.condition.label
  }
}

/**
 * Names of the overlays (melhorias + material) applied to an inventory row —
 * what the row shows as chips so upgrades are visible without opening the
 * overlay dialog. Unknown ids are skipped.
 *
 * @example itemOverlayNames(couracaRow) // ["Reforçada", "Aço-rubi"]
 */
export function itemOverlayNames(item: {
  improvements: string
  material: string | null
}): string[] {
  return itemOverlayCatalogs(item).map((o) => o.name)
}

/**
 * The resolved catalog entries of an item's overlays (melhorias + material),
 * for views that need more than the name — the bag sheet lists each overlay
 * with its effects. Unknown ids are skipped.
 *
 * @example itemOverlayCatalogs(couracaRow)[0].name // "Reforçada"
 */
export function itemOverlayCatalogs(item: {
  improvements: string
  material: string | null
}): CatalogItem[] {
  const ids = parseOverlayIds(item.improvements)
  if (item.material) ids.push(item.material)
  return ids
    .map((id) => getCatalogItem(id))
    .filter((o): o is CatalogItem => Boolean(o))
}

/** Defensive parse of the improvements JSON blob (bad blob ⇒ none). */
function parseOverlayIds(raw: string): string[] {
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is string => typeof x === 'string')
  } catch {
    return []
  }
}

/** Number formatter shared between inventory + catalog dialogs. */
export function formatLoad(n: number): string {
  if (Number.isInteger(n)) return String(n)
  return n.toFixed(1).replace('.', ',')
}

/**
 * One-line summary of an overlay's modifier notes for the melhoria/material
 * pickers. Deduplicated: Equilibrada carries four maneuver modifiers that
 * share the note "+2 em manobras" — joining them raw rendered
 * "+2 em manobras, +2 em manobras, +2 em manobras, +2 em manobras".
 *
 * @example overlayNotesSummary(equilibrada.modifiers) // "+2 em manobras"
 */
export function overlayNotesSummary(modifiers: readonly Modifier[]): string {
  const notes = modifiers.map((m) => m.note ?? '').filter(Boolean)
  return [...new Set(notes)].join(', ')
}

/**
 * Resolved carga-limit caption — computed limit + the For value that feeds
 * it, never math notation like "10 + 2×|FOR|" (bug F).
 *
 * @example loadLimitLabel(18, 4) // "limite 18 · 10 + 2×For +4"
 */
export function loadLimitLabel(max: number, forValue: number): string {
  return `limite ${max} · 10 + 2×For ${signed(forValue)}`
}
