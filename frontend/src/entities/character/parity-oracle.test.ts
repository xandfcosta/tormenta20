/// <reference types="node" />
// Dev harness: runs under vitest (Node), not shipped in the app bundle — the
// reference pulls in the already-installed @types/node without adding node to
// the app tsconfig's `types`.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
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
import { activeItemsFor, allConditionals, characterEffects } from './derived'
import { equippedItemFlagEffects } from './effect-source'
import { assembleWeaponCards } from './weapon-cards'
import { buildVitalContext } from './level-vitals'
import { computeVitalPools } from './vital-pools'

/**
 * PARITY HARNESS (PORT-PLAN.md §3) — the TDD backbone for the Go engine port.
 *
 * For each of the 18 seed characters, dumps the CURRENT `derived.ts` output as
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
 * Every conditional opt-in the character could toggle, in a stable order.
 *
 * The oracles ran with this set EMPTY on all 16 seeds, so `applyActiveConditionals`
 * — which folds an opt-in in and RE-RUNS `resolveStack` per target — had never
 * been exercised on real data by any golden, on either engine (ALE-106). Three
 * seeds carry conditionals; the richest is `bardo-versatil-nv7`, whose two
 * Inspiração entries hit the SAME target with the SAME bonusType at +1 and +2,
 * so turning both on must give +2 and not +3.
 *
 * For a character with no opt-ins this is identical to the base sheet — which is
 * itself worth having, since it pins the fold as a no-op when nothing is on.
 */
function conditionalIdsFor(char: Character): string[] {
  return allConditionals(char, EMPTY_CONDITIONALS)
    .map((entry) => entry.id)
    .sort()
}

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

/**
 * The golden payload for one seed character: everything the Go engine must
 * reproduce. `char` is included so the Go collection-layer test can re-run
 * `ActiveItemsFor` on the same raw input.
 */
function oraclePayloadFor(slug: string, char: Character) {
  const effects = characterEffects(char, EMPTY_CONDITIONALS)
  const conditionalIds = conditionalIdsFor(char)
  const withConditionals = characterEffects(char, new Set(conditionalIds))
  return {
    slug,
    char,
    activeItems: activeItemsFor(char),
    itemEffects: normalizeEffects(effects),
    sheetV2: sheetV2For(char),
    // The same sheet with every opt-in toggled ON — the only golden coverage of
    // the conditional fold (ALE-106). `activeConditionals` is dumped so the Go
    // mirror toggles exactly the same ids instead of re-deriving them.
    activeConditionals: conditionalIds,
    sheetV2WithConditionals: assembleSheetV2(char, withConditionals),
    weaponCardsWithConditionals: assembleWeaponCards(char, withConditionals),
    vitals: computeVitalPools(buildVitalContext(char, effects, char.classes)),
    // Equipped-item flag provenance oracle (Fase A.3.3) — label dropped so it
    // mirrors the Go `ComputeEquippedFlags` ({flag, source}) shape.
    equippedFlags: equippedItemFlagEffects(char.items).map((e) => ({
      flag: e.flag,
      source: e.source,
    })),
    // Wielded-weapon formula cards oracle (WeaponFormulaCards port).
    weaponCards: assembleWeaponCards(char, effects),
  }
}

/** Round-trip through JSON so the comparison sees what the file can hold. */
const serialized = (value: unknown) => JSON.parse(JSON.stringify(value))

function readOracle(file: string): unknown {
  try {
    return JSON.parse(readFileSync(resolve(oracleDir, file), 'utf8'))
  } catch {
    throw new Error(
      `oráculo de paridade ausente: engine-go/parity/${file}. ` +
        'Gere com `GEN_ORACLE=1 pnpm --filter frontend test parity-oracle`.',
    )
  }
}

describe('parity oracle — derived.ts golden output for the 18 seed chars', () => {
  /**
   * Without GEN_ORACLE this COMPARES instead of just smoke-checking (ALE-100).
   *
   * It used to assert only `chars.length === 16` and that two fields were the
   * right JS type, which made the oracle a photo rather than a mirror: change a
   * TS rule, and the committed JSONs silently went stale while the Go parity
   * tests stayed green against the old snapshot. Nothing in the repo — and
   * nothing in CI, which never sets GEN_ORACLE — could answer "are the oracles
   * up to date?" without overwriting them.
   */
  it('bate com o oráculo commitado de cada personagem da seed', () => {
    expect(chars).toHaveLength(18)
    const shouldWrite = !!process.env.GEN_ORACLE
    if (shouldWrite) mkdirSync(oracleDir, { recursive: true })

    for (const { slug, char } of chars) {
      const payload = oraclePayloadFor(slug, char)

      if (shouldWrite) {
        writeFileSync(
          resolve(oracleDir, `${slug}.json`),
          `${JSON.stringify(payload, null, 2)}\n`,
        )
        continue
      }

      expect(serialized(payload), `oráculo desatualizado para "${slug}"`).toEqual(
        readOracle(`${slug}.json`),
      )
    }
  })

  // Underscore-prefixed so the Go per-slug loops (which glob `*.json`) skip it.
  it('bate com o dump de catálogos que a camada de coleta lê', () => {
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
    const body = `${JSON.stringify(catalogs, null, 2)}\n`

    if (process.env.GEN_ORACLE) {
      mkdirSync(oracleDir, { recursive: true })
      writeFileSync(resolve(oracleDir, '_catalogs.json'), body)
      return
    }

    // Compared as text, not with toEqual: this dump is ~400 KB and a structural
    // diff on failure would bury the terminal. The actionable message is the
    // same either way — regenerate.
    const committed = readFileSync(resolve(oracleDir, '_catalogs.json'), 'utf8')
    expect(
      body.length,
      'o dump de catálogos mudou — rode `GEN_ORACLE=1 pnpm --filter frontend test parity-oracle`',
    ).toBe(committed.length)
    expect(body === committed, 'o dump de catálogos divergiu do commitado').toBe(true)
  })
})
