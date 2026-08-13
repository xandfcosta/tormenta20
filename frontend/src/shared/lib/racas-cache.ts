import { parseOrigemItem } from '@/shared/rules/origem-item-parse'
import type { Origem, OrigemItemGrant, Raca } from '@/shared/api/catalog-types'

/**
 * Front-owned cache for the racas.ts RACAS + origens.ts ORIGENS reference data
 * (character-build racial/origin tables), with SYNC accessors mirroring
 * t20-data's `racaById`/`racasByTier`/`origemById`. Lets derived.ts + the build
 * helpers read them WITHOUT a build-time `import` of the ~20KB tables — fetched
 * from `GET /catalog/races` + `/catalog/origens` and cached instead
 * (project_front_decouple_catalog). Same prime-before-render contract as the
 * other *-cache modules. O `resolveAtributoMod` puro vem de `shared/rules`, para
 * onde mudou quando o `t20-data` foi apagado (ALE-109) — data-free, tree-shaka
 * sozinho.
 */
let racasById: Readonly<Record<string, Raca>> = {}
let racasArr: readonly Raca[] = []
let origensById: Readonly<Record<string, Origem>> = {}
let origensArr: readonly Origem[] = []
let primed = false

export function primeRacas(
  racas: Readonly<Record<string, Raca>>,
  origens: Readonly<Record<string, Origem>>,
): void {
  racasById = racas
  racasArr = Object.values(racas)
  origensById = origens
  origensArr = Object.values(origens)
  primed = true
}

/** True once the racas/origens cache has been primed — for a render-time gate. */
export function isRacasPrimed(): boolean {
  return primed
}

/** Cache-backed mirror of t20-data `racaById` — THROWS on unknown id to match. */
export function racaById(id: string): Raca {
  const r = racasById[id]
  if (!r) throw new Error(`racaById: unknown raça id "${id}"`)
  return r
}

/** Raças of a tier — mirrors t20-data `racasByTier`. */
export function racasByTier(tier: 'comum' | 'extra'): Raca[] {
  return racasArr.filter((r) => r.tier === tier)
}

/** The full raça list / record (were `Object.values(RACAS)` / `RACAS`). Read
 *  inside functions that run AFTER the gate, not at module top-level. */
export function racasList(): readonly Raca[] {
  return racasArr
}
export function racasRecord(): Readonly<Record<string, Raca>> {
  return racasById
}

/** Cache-backed mirror of t20-data `origemById` — THROWS on unknown id. */
export function origemById(id: string): Origem {
  const o = origensById[id]
  if (!o) throw new Error(`origemById: unknown origem id "${id}"`)
  return o
}

export function origensList(): readonly Origem[] {
  return origensArr
}
export function origensRecord(): Readonly<Record<string, Origem>> {
  return origensById
}

/** Structured starting-item grants for an origem, by NAME. Cache-backed mirror
 *  of t20-data `origemItemGrantsByName` (pure parser, cached origens). */
export function origemItemGrantsByName(originName: string): OrigemItemGrant[] {
  const origem = origensArr.find((o) => o.name === originName)
  return (origem?.itensIniciais ?? []).map(parseOrigemItem)
}
