import {
  computeItemEffects,
  type DisplayFact,
  type ItemFlag,
} from '@tormenta20/t20-data'
import { getActivation } from '@/shared/lib/activation-cache'
import { getCatalogItem } from '@/shared/lib/catalog-cache'
import { spellCatalog } from '@/shared/lib/spell-cache'
import {
  computeEquippedFlags as engineComputeEquippedFlags,
  type EquippedItemFlag,
} from '@/shared/lib/engine-wasm'
import type { CharacterItem } from '@/shared/api/api'

/**
 * Display name for an ActiveEffect's source id. Item-sourced effects resolve
 * through the item catalog; spell buffs (Phase-1 `SpellBuff`) through the spell
 * catalog; anything else falls back to the raw id. `getCatalogItem` returns
 * undefined for a spell id, which is exactly the case this bridges.
 */
export function effectSourceName(catalogId: string): string {
  // GM-entered ad-hoc pool (F3) — no catalog entry behind it.
  if (catalogId === 'manual-temp-hp') return 'PV temporários (manual)'
  return (
    getCatalogItem(catalogId)?.name ??
    spellCatalog()[catalogId]?.name ??
    // Power-granted effects persist the power id as catalogId (Fase 4) —
    // "class.barbaro.alma-de-bronze" must read "Alma de Bronze".
    getActivation(catalogId)?.name ??
    catalogId
  )
}

/**
 * Display-only facts (RD, immunities, senses, …) for a spell-sourced effect —
 * so an applied buff can surface its non-computed sub-effects as reference
 * chips. Empty for item sources / unknown ids.
 */
export function effectSourceFacts(catalogId: string): DisplayFact[] {
  return spellCatalog()[catalogId]?.buff?.facts ?? []
}

/**
 * pt-BR labels for the engine's ItemFlag names, so 'fatigue-on-sleep' always
 * reads "Fadiga ao dormir". Exhaustive Record — a new ItemFlag in t20-data
 * fails typecheck here instead of rendering a raw id. Strings are kept
 * byte-identical to `ITEM_FLAG_LABELS` in features/character-sheet/
 * item-describe.ts (which this entities module cannot import under FSD);
 * consolidate there by re-exporting from here when that file is next touched.
 */
export const ITEM_FLAG_LABEL: Record<ItemFlag, string> = {
  'lethal-unarmed': 'Ataques desarmados causam dano letal',
  'cannot-apply-dex-to-defense': 'Não soma Destreza na Defesa',
  'fatigue-on-sleep': 'Fadiga ao dormir',
  'reach-extends': 'Alcance ampliado',
  'armadura-pesada': 'Conta como armadura pesada',
}

export type ItemFlagEffect = {
  flag: ItemFlag
  label: string
  source: string
}

/**
 * Always-on flag effects carried by equipped items (heavy armor's
 * fatigue-on-sleep / armadura-pesada, …) with item provenance, for read-only
 * display in the Efeitos tab. Same MODE-gate as the other engine choke points:
 * production/dev runs the Go/WASM `ComputeEquippedFlags`; the TS branch below is
 * TEST-ONLY (parity oracle, no wasm in vitest) and is dead-code-eliminated from
 * the app bundle — taking `computeItemEffects` (the last effects-cru consumer)
 * with it. The pt-BR label is always mapped on this side.
 *
 * Usage: `equippedItemFlagEffects(character.items)` → `[{ flag, label, source }]`.
 */
export function equippedItemFlagEffects(
  items: readonly CharacterItem[],
): ItemFlagEffect[] {
  const raw = engineComputeEquippedFlags(items)
  return raw.map((f) => ({
    flag: f.flag as ItemFlag,
    label: ITEM_FLAG_LABEL[f.flag as ItemFlag],
    source: f.source,
  }))
}

/**
 * A IMPLEMENTAÇÃO DE REFERÊNCIA das bandeiras, que gera o oráculo. Chamada
 * explicitamente pelo harness, nunca por `if` de ambiente — atravessar o choke
 * point faria o oráculo virar "o Go dizendo o que o Go acha" (ALE-109).
 *
 * Per equipped catalog item, run
 * ONLY its base catalog modifiers through the item engine (so wear conditions
 * resolve like the sheet totals) and collect the flags, sorted per item for a
 * deterministic order matching the Go `ComputeEquippedFlags`.
 */
export function tsEquippedItemFlags(items: readonly CharacterItem[]): EquippedItemFlag[] {
  const out: EquippedItemFlag[] = []
  for (const it of items) {
    if (it.equipped === null || !it.catalogId) continue
    const modifiers = getCatalogItem(it.catalogId)?.modifiers ?? []
    const { flags } = computeItemEffects([
      { source: it.name, equipped: it.equipped, modifiers },
    ])
    for (const flag of [...flags].sort()) {
      out.push({ flag, source: it.name })
    }
  }
  return out
}
