import type {
  Condition,
  ConditionId,
  TormentaPower,
  TormentaPowerId,
} from '@tormenta20/t20-data'

/**
 * Front-owned cache for the rules reference records CONDITIONS (conditions.ts)
 * + TORMENTA_POWERS (tormenta.ts), with SYNC accessors so the sheet/derive/GM
 * UIs read them WITHOUT a build-time `import` of the ~12KB tables — fetched from
 * `GET /catalog/conditions` + `/catalog/tormenta-powers` and cached instead
 * (project_front_decouple_catalog). Same prime-before-render contract as the
 * other *-cache modules.
 */
let conditionsRec: Readonly<Record<ConditionId, Condition>> =
  {} as Record<ConditionId, Condition>
let conditionsArr: readonly Condition[] = []
let tormentaRec: Readonly<Record<TormentaPowerId, TormentaPower>> =
  {} as Record<TormentaPowerId, TormentaPower>
let primed = false

export function primeRulesCatalogs(
  conditions: Readonly<Record<ConditionId, Condition>>,
  tormentaPowers: Readonly<Record<TormentaPowerId, TormentaPower>>,
): void {
  conditionsRec = conditions
  conditionsArr = Object.values(conditions)
  tormentaRec = tormentaPowers
  primed = true
}

/** True once the rules catalogs have been primed — for a render-time gate. */
export function isRulesCatalogPrimed(): boolean {
  return primed
}

/** The CONDITIONS record (was the t20-data `CONDITIONS` const) — for `id in`
 *  membership + `CONDITIONS[id]` index. Read after the gate. */
export function conditionsRecord(): Readonly<Record<ConditionId, Condition>> {
  return conditionsRec
}
export function conditionsList(): readonly Condition[] {
  return conditionsArr
}

/** The TORMENTA_POWERS record (was the t20-data `TORMENTA_POWERS` const) — for
 *  `id in` membership + `TORMENTA_POWERS[id]` index. Read after the gate. */
export function tormentaPowersRecord(): Readonly<
  Record<TormentaPowerId, TormentaPower>
> {
  return tormentaRec
}
