import type {
  AttributeKey,
  BonusType,
  ItemEffects,
  ModifierTarget,
} from '@tormenta20/t20-data'
import type { Character } from '@/shared/api/api'
import type { ComputedSheetV2, WeaponCard } from './computed-sheet-v2'

/**
 * Normalized input for the Go vitals pipeline (PV/PM máximos) — mirrors t20-data
 * VitalGrantContext. Built by the front consumers (level-vitals, draft-vitals).
 * `raceId` is the race NAME (getRace resolves by it); `attrTotals` are the FINAL
 * totals (post item/race mods).
 */
export type VitalContext = {
  level: number
  classes: { className: string; level: number }[]
  raceId: string
  raceAbilityChoices: string[]
  powerIds: string[]
  classChoices: Record<string, { devoto?: string; caminho?: string } | undefined>
  godPower: string
  origin: string
  originChoices: string[]
  attrTotals: Record<AttributeKey, number>
}

export type VitalPools = { pvMax: number; pmMax: number }

/**
 * Lazy loader + typed wrapper for the engine-go rules engine compiled to WASM.
 * Runs the SAME Go rules the server runs — single source
 * of truth, no TS derive duplication (project_front_decouple_catalog Fase 3:
 * compute is free, WASM ~0.28ms/sheet + a one-time 0.95MB load, and works
 * offline). Artifacts are built by `scripts/build-engine-wasm.sh` into
 * `public/engine/` and served by Vite at `/engine/*`.
 *
 * The Go runtime glue (`wasm_exec.js`) defines `globalThis.Go`; we inject it as
 * a <script> (CSP-friendly, no eval) then instantiate the module once. Call
 * `ensureEngine()` from the route loader (parallel to `ensureCatalogs`) so the
 * engine is warm before the first derive; `computeSheet` is sync afterwards.
 */
type GoRuntime = {
  importObject: WebAssembly.Imports
  run(instance: WebAssembly.Instance): void
}
type GlobalWithGo = typeof globalThis & {
  Go?: new () => GoRuntime
  primeEngineCatalogs?: (payloadJson: string) => string
  computeSheetV2?: (charJson: string, conditionalsJson: string) => string
  computeEffects?: (charJson: string, conditionalsJson: string) => string
  computeVitals?: (contextJson: string) => string
  resolveConditionalDisplay?: (rowsJson: string) => string
  computeEquippedFlags?: (itemsJson: string) => string
  pointBuyStatus?: (attrsJson: string) => string
  computeWeaponCards?: (charJson: string, conditionalsJson: string) => string
  spellPmLimit?: (charJson: string, spellClassesJson: string) => string
}

/** Creation point-buy result (p17): total spent (null when a value is out of
 *  range) + advisory warnings — mirrors the Go `PointBuyStatus`. */
export type PointBuyStatus = { spent: number | null; warnings: string[] }

/** One always-on flag carried by an equipped item, with item provenance —
 *  mirrors the Go `EquippedFlag` (the pt-BR label is added on the entities side). */
export type EquippedItemFlag = { flag: string; source: string }

/** One conditional-effect row (an active stance's modifier) fed to the display
 *  resolver — mirrors the Go `ConditionalDisplayInput`. */
export type ConditionalDisplayInput = {
  target: ModifierTarget
  bonusType: BonusType
  amount: number
}

/** A surviving {target, amount} row after non-stacking display resolution. */
export type ConditionalDisplayRow = { target: ModifierTarget; amount: number }

/** The engine's serialized ItemEffects — flags as a sorted array (rebuilt into a
 *  Set on the TS side to match the t20-data `ItemEffects` shape). */
type SerializedEffects = {
  byTarget: ItemEffects['byTarget']
  flags: string[]
  conditional: ItemEffects['conditional']
  error?: string
}

let catalogsPrimed = false

const WASM_URL = '/engine/t20.wasm'
const GLUE_URL = '/engine/wasm_exec.js'
let enginePromise: Promise<void> | null = null

function loadGlue(): Promise<void> {
  const g = globalThis as GlobalWithGo
  if (g.Go) return Promise.resolve()
  return new Promise<void>((resolve, reject) => {
    const s = document.createElement('script')
    s.src = GLUE_URL
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`engine-wasm: failed to load ${GLUE_URL}`))
    document.head.appendChild(s)
  })
}

async function initEngine(): Promise<void> {
  await loadGlue()
  const g = globalThis as GlobalWithGo
  if (!g.Go) throw new Error('engine-wasm: wasm_exec.js did not define globalThis.Go')
  const go = new g.Go()
  const { instance } = await WebAssembly.instantiateStreaming(
    fetch(WASM_URL),
    go.importObject,
  )
  go.run(instance) // main() blocks on select{}; registers the engine globals
}

/** Load + instantiate the WASM engine once (cached). Warm it from the route
 *  loader before rendering, like `ensureCatalogs`. */
export function ensureEngine(): Promise<void> {
  enginePromise ??= initEngine()
  return enginePromise
}

/**
 * Prime the REAL derive's catalogs once — the same JSON `ensureCatalogs` fetches
 * (items/races/origins/classPowers/generalPowers/racas/tormentaPowerIds).
 * Required before `computeSheetV2`; idempotent (re-priming replaces the set).
 * Requires `ensureEngine()` to have resolved.
 */
export function primeEngineCatalogs(payloadJson: string): void {
  const fn = (globalThis as GlobalWithGo).primeEngineCatalogs
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const res = JSON.parse(fn(payloadJson)) as { ok?: boolean; error?: string }
  if (res.error) throw new Error(`engine-wasm: ${res.error}`)
  catalogsPrimed = true
}

/** True once the engine catalogs have been primed — for a render-time gate. */
export function areEngineCatalogsPrimed(): boolean {
  return catalogsPrimed
}

/**
 * Compute the resolved `ItemEffects` through the Go engine over a RAW Character +
 * the active-conditional ids — the sheet derive's choke point (Inc.2 task #7).
 * The heavy collection + resolution runs in Go; the thin `derived.ts` breakdown
 * helpers stay TS over these effects. Requires `ensureEngine()` +
 * `primeEngineCatalogs()`. Mirrors `characterEffects` byte-for-byte (verified by
 * the `itemEffects` parity oracle). Flags come back as an array → rebuilt into a
 * Set to match the t20-data `ItemEffects` shape.
 */
export function computeEffects(
  char: Character,
  conditionals: readonly string[] = [],
): ItemEffects {
  const fn = (globalThis as GlobalWithGo).computeEffects
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(
    fn(JSON.stringify(char), JSON.stringify(conditionals)),
  ) as SerializedEffects
  if (out.error) throw new Error(`engine-wasm: ${out.error}`)
  return {
    byTarget: out.byTarget,
    flags: new Set(out.flags) as ItemEffects['flags'],
    conditional: out.conditional,
  }
}

/**
 * Compute PV/PM máximos through the Go engine over a normalized `VitalContext`.
 * Requires `ensureEngine()` + `primeEngineCatalogs()`. Mirrors the front's TS
 * vitals pipeline byte-for-byte (verified by the `vitals` parity oracle).
 */
export function computeVitals(context: VitalContext): VitalPools {
  const fn = (globalThis as GlobalWithGo).computeVitals
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(context))) as VitalPools & { error?: string }
  if (out.error) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * Compute the rich derived sheet (every breakdown) through the Go engine over a
 * RAW Character + the active-conditional ids. Requires `ensureEngine()` +
 * `primeEngineCatalogs()`. Mirrors the front's `derived.ts` breakdowns
 * byte-for-byte (verified by the `sheetV2` parity oracle).
 */
export function computeSheetV2(
  char: Character,
  conditionals: readonly string[] = [],
): ComputedSheetV2 {
  const fn = (globalThis as GlobalWithGo).computeSheetV2
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(
    fn(JSON.stringify(char), JSON.stringify(conditionals)),
  ) as ComputedSheetV2 & { error?: string }
  if (out.error) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * Non-stacking display resolution over an active stance's conditional rows (Go
 * `ResolveConditionalDisplay`) — pure, needs no primed catalogs, only the loaded
 * engine. Mirrors t20-data `resolveConditionalDisplay` byte-for-byte.
 */
export function resolveConditionalDisplay(
  rows: readonly ConditionalDisplayInput[],
): ConditionalDisplayRow[] {
  const fn = (globalThis as GlobalWithGo).resolveConditionalDisplay
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(rows))) as
    | ConditionalDisplayRow[]
    | { error: string }
  if (!Array.isArray(out)) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * Always-on flags of a character's equipped items with provenance (Go
 * `ComputeEquippedFlags`). Requires `ensureEngine()` + `primeEngineCatalogs()`
 * (item modifiers come from the catalog). Mirrors the TS `tsEquippedFlags`.
 */
export function computeEquippedFlags(
  items: readonly unknown[],
): EquippedItemFlag[] {
  const fn = (globalThis as GlobalWithGo).computeEquippedFlags
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(items))) as
    | EquippedItemFlag[]
    | { error: string }
  if (!Array.isArray(out)) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * Creation point-buy status (p17) over a base attribute spread (Go
 * `PointBuyStatusFor`) — pure, needs no primed catalogs, only the loaded engine.
 * Mirrors t20-data `pointBuySpent` + `pointBuyWarnings`.
 */
export function pointBuyStatus(attrs: Record<string, number>): PointBuyStatus {
  const fn = (globalThis as GlobalWithGo).pointBuyStatus
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(attrs))) as PointBuyStatus & { error?: string }
  if (out.error) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * Wielded-weapon formula cards (attack/damage/crit) through the Go engine over a
 * RAW Character + active-conditional ids. Requires `ensureEngine()` +
 * `primeEngineCatalogs()`. Mirrors the front's `assembleWeaponCards` byte-for-byte
 * (verified by the `weaponCards` parity oracle).
 */
export function computeWeaponCards(
  char: Character,
  conditionals: readonly string[] = [],
): WeaponCard[] {
  const fn = (globalThis as GlobalWithGo).computeWeaponCards
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(
    fn(JSON.stringify(char), JSON.stringify(conditionals)),
  ) as WeaponCard[] | { error: string }
  if (!Array.isArray(out)) throw new Error(`engine-wasm: ${out.error}`)
  return out
}

/**
 * The p224 PM ceiling for ONE spell: the level in the class that grants it, or
 * the character level when the source is not a class.
 *
 * This is NOT `sheet.pmLimit.total` — that one is the per-character HUD summary
 * ("best caster level"). The cast dialog used to gate on the summary, so a
 * Bardo 7 / Arcanista 1 was offered 7 PM on an Arcanista spell and the server
 * refused anything over 1 (ALE-92). Same Go function the cast handler runs.
 *
 * @example spellPmLimit(character, spell.classes) // 1
 */
export function spellPmLimit(char: Character, spellClasses: readonly string[]): number {
  const fn = (globalThis as GlobalWithGo).spellPmLimit
  if (!fn) throw new Error('engine-wasm: engine not ready — call ensureEngine() first')
  const out = JSON.parse(fn(JSON.stringify(char), JSON.stringify(spellClasses))) as
    | { limit: number }
    | { error: string }
  if ('error' in out) throw new Error(`engine-wasm: ${out.error}`)
  return out.limit
}
