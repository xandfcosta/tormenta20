import type { CatalogSpell } from '@tormenta20/t20-data'
import { normalizeText } from '@/shared/lib/normalize-text'

/**
 * Front-owned spell catalog cache with SYNC accessors mirroring t20-data's
 * (`spellById`, `spellByName`, `spellEffectByName`). Lets the core derive
 * pipeline (effect-source.ts → derived.ts/temp-hp-pool.ts) and the spell UI
 * read the catalog WITHOUT a build-time `import` of the ~175KB SPELL_CATALOG —
 * it's fetched from `GET /catalog/spells` and cached instead
 * (project_front_decouple_catalog A).
 *
 * Same contract as `catalog-cache` / `abilities-cache`: correctness rests on
 * the cache being primed (via the root loader's `ensureCatalogs`) BEFORE any
 * consumer renders/derives. The catalog is static + cached-forever, so by
 * render time the primed value is effectively constant.
 */
let byId: Readonly<Record<string, CatalogSpell>> = {}
let byNormalizedName: ReadonlyMap<string, CatalogSpell> = new Map()
let effectByName: ReadonlyMap<string, string> = new Map()
let primed = false

/** Prime the spell cache from the fetched id-keyed catalog. Idempotent. */
export function primeSpellCatalog(
  catalog: Readonly<Record<string, CatalogSpell>>,
): void {
  byId = catalog
  const spells = Object.values(catalog)
  // Accent/case-insensitive name lookup, mirroring t20-data's spellByName.
  byNormalizedName = new Map(spells.map((s) => [normalizeText(s.name), s]))
  effectByName = new Map(spells.map((s) => [s.name, s.baseEffect]))
  primed = true
}

/** True once the spell cache has been primed — for a render-time gate. */
export function isSpellCatalogPrimed(): boolean {
  return primed
}

/** The raw id-keyed catalog record (was the t20-data `SPELL_CATALOG` const).
 *  Read inside components/functions that run AFTER the gate, never at module
 *  top-level (which evaluates before priming). */
export function spellCatalog(): Readonly<Record<string, CatalogSpell>> {
  return byId
}

/** Cache-backed mirror of t20-data `spellById` — THROWS on unknown id to match
 *  the source contract (consumers guard with try/catch). */
export function spellById(id: string): CatalogSpell {
  const spell = byId[id]
  if (!spell) throw new Error(`spellById: unknown spell id "${id}"`)
  return spell
}

/** Non-throwing existence check — for the `validateLearnSpell` lookup arg. */
export function hasSpell(id: string): boolean {
  return byId[id] !== undefined
}

/** Catalog spell by display name, accent-insensitive; null when none matches.
 *  Mirrors t20-data `spellByName`. */
export function spellByName(name: string): CatalogSpell | null {
  return byNormalizedName.get(normalizeText(name)) ?? null
}

/** Base-effect blurb of a spell by display name, or null. Mirrors t20-data
 *  `spellEffectByName`. */
export function spellEffectByName(name: string): string | null {
  return effectByName.get(name) ?? null
}

/** Every catalog spell carrying a `buff` block (Phase-1 SpellBuff) — the
 *  applicable-as-ActiveEffect list (was `Object.values(SPELL_CATALOG).filter`
 *  module-consts in the effects/initiative UIs). */
export function buffSpells(): CatalogSpell[] {
  return Object.values(byId).filter((s) => s.buff)
}
