import type { Modifier } from '@tormenta20/t20-data';

/**
 * Pure temp-PV pool math (livro p256: PV temporários não acumulam — vale o
 * maior). The services parse ActiveEffect rows into pools here, plan the
 * writes, and only then touch Prisma, so every rule is unit-testable without
 * I/O.
 */

/** The ActiveEffect columns the pool rules read. */
export type TempHpEffectRow = {
  id: number;
  catalogId: string;
  scope: string;
  modifiers: string;
};

export type ParsedTempHpPool = {
  effectId: number;
  catalogId: string;
  scope: string;
  amount: number;
  /** true when EVERY modifier targets tempHp — safe to delete the whole row.
   *  Mixed rows (Heroísmo: tempHp + attack/damage) only get their pool zeroed. */
  pure: boolean;
  modifiers: Modifier[];
};

function modifiersOf(raw: string): Modifier[] {
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as Modifier[]) : [];
  } catch {
    return [];
  }
}

/**
 * Extract the live temp-PV pools (amount > 0) from a character's effect rows.
 *
 * @example parseTempHpPools(rows)[0] // { effectId: 5, amount: 10, pure: true, … }
 */
export function parseTempHpPools(
  rows: readonly TempHpEffectRow[],
): ParsedTempHpPool[] {
  const pools: ParsedTempHpPool[] = [];
  for (const row of rows) {
    const modifiers = modifiersOf(row.modifiers);
    const tempHp = modifiers.find((m) => m?.target?.k === 'tempHp');
    if (!tempHp || tempHp.amount <= 0) continue;
    pools.push({
      effectId: row.id,
      catalogId: row.catalogId,
      scope: row.scope,
      amount: tempHp.amount,
      pure: modifiers.every((m) => m?.target?.k === 'tempHp'),
      modifiers,
    });
  }
  return pools;
}

/** Rewrite the tempHp modifier of a pool to a new amount (0 = drained). */
export function withTempHpAmount(
  modifiers: readonly Modifier[],
  amount: number,
): Modifier[] {
  return modifiers.map((m) =>
    m?.target?.k === 'tempHp' ? { ...m, amount } : m,
  );
}

/** A pool displaced by a bigger one: removed rows are deleted, kept (mixed)
 *  rows had their tempHp amount zeroed. Mirrors the client cache patch. */
export type DisplacedPool = { effectId: number; removed: boolean };

export type PoolSupremacyPlan =
  | { kind: 'superseded'; keptEffectId: number; keptAmount: number }
  | {
      kind: 'apply';
      displaced: DisplacedPool[];
      /** Mixed rows to persist with their pool zeroed. */
      zeroWrites: { effectId: number; modifiers: string }[];
      /** Pure pool rows to delete outright. */
      deleteIds: number[];
    };

/**
 * Vale-o-maior (p256): decide whether a new pool of `newAmount` from
 * `own` (catalogId+scope, replaced via upsert regardless) may exist next to
 * the character's other pools. An existing bigger-or-equal pool wins
 * (superseded, nothing written); otherwise the new pool wins and every
 * smaller pool is displaced.
 *
 * @example planPoolSupremacy(pools, { catalogId: 'manual-temp-hp', scope: 'scene' }, 12)
 */
export function planPoolSupremacy(
  pools: readonly ParsedTempHpPool[],
  own: { catalogId: string; scope: string },
  newAmount: number,
): PoolSupremacyPlan {
  const others = pools.filter(
    (p) => !(p.catalogId === own.catalogId && p.scope === own.scope),
  );
  const top = others.reduce<ParsedTempHpPool | null>(
    (best, p) => (best && best.amount >= p.amount ? best : p),
    null,
  );
  if (top && top.amount >= newAmount) {
    return { kind: 'superseded', keptEffectId: top.effectId, keptAmount: top.amount };
  }
  return {
    kind: 'apply',
    displaced: others.map((p) => ({ effectId: p.effectId, removed: p.pure })),
    zeroWrites: others
      .filter((p) => !p.pure)
      .map((p) => ({
        effectId: p.effectId,
        modifiers: JSON.stringify(withTempHpAmount(p.modifiers, 0)),
      })),
    deleteIds: others.filter((p) => p.pure).map((p) => p.effectId),
  };
}

/** One drained pool in a damage application. `removed` = row deleted (pure
 *  pool emptied); kept rows are persisted with `newAmount` (0 for an emptied
 *  mixed row, partial otherwise). */
export type DamageDrainStep = {
  effectId: number;
  newAmount: number;
  removed: boolean;
};

export type DamagePlan = {
  drained: DamageDrainStep[];
  /** Rows to persist with rewritten modifiers (partial drains + emptied mixed). */
  updates: { effectId: number; modifiers: string }[];
  deleteIds: number[];
  hpCurrent: number;
  tempHpRemaining: number;
};

/**
 * Route `amount` of damage temp-first: pools drain highest first (with
 * vale-o-maior there is normally a single pool; legacy multi-pool rows drain
 * biggest-shield-first), the remainder lowers hpCurrent with a floor of 0.
 *
 * @example planDamage(pools, 20, 7) // pool 10 → 3, hp untouched
 */
export function planDamage(
  pools: readonly ParsedTempHpPool[],
  hpCurrent: number,
  amount: number,
): DamagePlan {
  const plan: DamagePlan = {
    drained: [],
    updates: [],
    deleteIds: [],
    hpCurrent,
    tempHpRemaining: 0,
  };
  let left = amount;
  for (const pool of [...pools].sort((a, b) => b.amount - a.amount)) {
    const drained = Math.min(left, pool.amount);
    left -= drained;
    drainPool(plan, pool, pool.amount - drained);
  }
  plan.hpCurrent = Math.max(0, hpCurrent - left);
  return plan;
}

function drainPool(
  plan: DamagePlan,
  pool: ParsedTempHpPool,
  newAmount: number,
): void {
  plan.tempHpRemaining += newAmount;
  if (newAmount === pool.amount) return; // untouched — not part of the delta
  const removed = newAmount === 0 && pool.pure;
  plan.drained.push({ effectId: pool.effectId, newAmount, removed });
  if (removed) {
    plan.deleteIds.push(pool.effectId);
    return;
  }
  plan.updates.push({
    effectId: pool.effectId,
    modifiers: JSON.stringify(withTempHpAmount(pool.modifiers, newAmount)),
  });
}
