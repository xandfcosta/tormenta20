/// <reference types="node" />
// Dev harness: runs under vitest (Node), not shipped in the app bundle — the
// reference pulls in the already-installed @types/node without adding node to
// the app tsconfig's `types`.
import { mkdirSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  CATALOG_ITEMS,
  CLASS_POWERS_CATALOG,
  GENERAL_POWERS_CATALOG,
  GRANTED_POWERS,
  type ItemEffects,
  ORIGINS_CATALOG,
  RACAS,
  RACES_CATALOG,
  TORMENTA_POWERS,
} from '@tormenta20/t20-data'
import { describe, expect, it } from 'vitest'
import type { Character } from '@/shared/api/api'
import fixtures from './__fixtures__/character-input-parity.json'
import { assembleSheetV2 } from './computed-sheet'
import { activeItemsFor, characterEffects } from './derived'
import { buildVitalContext } from './level-vitals'
import { computeVitalPools } from './vital-pools'

/**
 * PARITY HARNESS (PORT-PLAN.md §3) — the TDD backbone for the Go engine port.
 *
 * For each of the 16 seed characters, dumps the CURRENT `derived.ts` output as
 * the golden oracle the Go engine (engine-go) is checked against:
 *   - `activeItems` — the collection-layer output; slice 2's target + the
 *     resolution engine's real-data input.
 *   - `itemEffects` — the resolution output (flags Set normalized to a sorted
 *     array so JSON parity is order-independent, matching the Go MarshalJSON).
 *   - `sheetV2` — every derived.ts breakdown (`*Total` + RD/tempHp), the shape
 *     the Go `ComputeSheetV2` must reproduce (slice 3 / task #5).
 *
 * The oracle files (`engine-go/parity/<slug>.json`) are committed. Regenerate
 * them whenever the TS rules change, while `derived.ts` is still the reference:
 *   GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
 */

const EMPTY_CONDITIONALS: ReadonlySet<string> = new Set()

/**
 * The full breakdown sheet the Go ComputeSheetV2 mirrors. Built from the shared
 * `assembleSheetV2` (which calls the real derived.ts breakdowns), so the oracle
 * generator and the `useComputedSheet` test branch stay the single source of
 * truth. `assembleSheetV2` uses furiaActive=true for `tempHpFuria` (the base
 * sheet with furia off is always {0, []}).
 */
function sheetV2For(char: Character) {
  return assembleSheetV2(char, characterEffects(char, EMPTY_CONDITIONALS))
}

/** Serializable ItemEffects: Set → sorted array, mirroring the Go MarshalJSON. */
function normalizeEffects(e: ItemEffects) {
  return {
    byTarget: e.byTarget,
    flags: [...e.flags].sort(),
    conditional: e.conditional,
  }
}

const oracleDir = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../../engine-go/parity',
)

const chars = fixtures as { slug: string; char: Character }[]

describe('parity oracle — derived.ts golden output for the 16 seed chars', () => {
  it('generates one oracle per seed character', () => {
    expect(chars).toHaveLength(16)
    const shouldWrite = !!process.env.GEN_ORACLE
    if (shouldWrite) mkdirSync(oracleDir, { recursive: true })

    for (const { slug, char } of chars) {
      const activeItems = activeItemsFor(char)
      const itemEffects = normalizeEffects(characterEffects(char, EMPTY_CONDITIONALS))
      expect(Array.isArray(activeItems)).toBe(true)
      expect(itemEffects.byTarget).toBeTypeOf('object')

      if (shouldWrite) {
        // `char` is included so the Go collection-layer test (slice 2) can
        // re-run `ActiveItemsFor` on the same raw input and check it against
        // `activeItems` — the resolution test (slice 1) only needs activeItems.
        // `sheetV2` is the breakdown oracle (task #5).
        const vitals = computeVitalPools(
          buildVitalContext(char, characterEffects(char, EMPTY_CONDITIONALS), char.classes),
        )
        const payload = {
          slug,
          char,
          activeItems,
          itemEffects,
          sheetV2: sheetV2For(char),
          vitals,
        }
        writeFileSync(
          resolve(oracleDir, `${slug}.json`),
          `${JSON.stringify(payload, null, 2)}\n`,
        )
      }
    }
  })

  // Underscore-prefixed so the Go per-slug loops (which glob `*.json`) skip it.
  it('dumps the catalogs the collection layer reads (for the Go engine)', () => {
    if (!process.env.GEN_ORACLE) return
    mkdirSync(oracleDir, { recursive: true })
    const catalogs = {
      items: CATALOG_ITEMS,
      races: RACES_CATALOG,
      origins: ORIGINS_CATALOG,
      classPowers: CLASS_POWERS_CATALOG,
      generalPowers: GENERAL_POWERS_CATALOG,
      grantedPowers: GRANTED_POWERS,
      racas: RACAS,
      tormentaPowerIds: Object.keys(TORMENTA_POWERS),
    }
    writeFileSync(
      resolve(oracleDir, '_catalogs.json'),
      `${JSON.stringify(catalogs, null, 2)}\n`,
    )
  })
})
