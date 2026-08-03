import {
  parseTempHpPools,
  planDamage,
  planPoolSupremacy,
  withTempHpAmount,
  type TempHpEffectRow,
} from './temp-hp.helpers';

/** Named fake rows — pure pool (only tempHp) vs mixed buff (Heroísmo). */
function fakePoolRow(
  id: number,
  amount: number,
  catalogId = 'class.barbaro.alma-de-bronze',
  scope = 'scene',
): TempHpEffectRow {
  return {
    id,
    catalogId,
    scope,
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'PV temporários' },
    ]),
  };
}

function fakeMixedRow(id: number, amount: number): TempHpEffectRow {
  return {
    id,
    catalogId: 'heroismo',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'tempHp' }, amount, bonusType: 'untyped', note: 'Heroísmo' },
      { target: { k: 'attack', scope: 'all' }, amount: 4, bonusType: 'untyped' },
    ]),
  };
}

function fakeDefenseRow(id: number): TempHpEffectRow {
  return {
    id,
    catalogId: 'armadura-arcana',
    scope: 'scene',
    modifiers: JSON.stringify([
      { target: { k: 'defense' }, amount: 5, bonusType: 'armor' },
    ]),
  };
}

describe('parseTempHpPools', () => {
  it('extracts amount + purity, skipping non-pool and drained rows', () => {
    const pools = parseTempHpPools([
      fakePoolRow(1, 10),
      fakeMixedRow(2, 40),
      fakeDefenseRow(3),
      fakePoolRow(4, 0),
    ]);
    expect(pools.map((p) => [p.effectId, p.amount, p.pure])).toEqual([
      [1, 10, true],
      [2, 40, false],
    ]);
  });

  it('tolerates malformed modifier blobs', () => {
    expect(
      parseTempHpPools([{ ...fakePoolRow(1, 5), modifiers: '{oops' }]),
    ).toEqual([]);
  });
});

describe('withTempHpAmount', () => {
  it('rewrites only the tempHp modifier', () => {
    const modifiers = [
      { target: { k: 'tempHp' }, amount: 40 },
      { target: { k: 'attack', scope: 'all' }, amount: 4 },
    ];
    const next = withTempHpAmount(modifiers as never, 0);
    expect(next[0]!.amount).toBe(0);
    expect(next[1]!.amount).toBe(4);
  });
});

describe('planPoolSupremacy — vale o maior (p256)', () => {
  const own = { catalogId: 'class.barbaro.alma-de-bronze', scope: 'scene' };

  it('same source+scope replaces its own pool (not compared against itself)', () => {
    const plan = planPoolSupremacy(
      parseTempHpPools([fakePoolRow(1, 99)]),
      own,
      10,
    );
    expect(plan).toEqual({
      kind: 'apply',
      displaced: [],
      zeroWrites: [],
      deleteIds: [],
    });
  });

  it('is a no-op (superseded) when an existing pool is bigger or equal', () => {
    const plan = planPoolSupremacy(
      parseTempHpPools([fakePoolRow(7, 30, 'campo-de-forca')]),
      own,
      10,
    );
    expect(plan).toEqual({ kind: 'superseded', keptEffectId: 7, keptAmount: 30 });
  });

  it('deletes smaller pure pools when the new pool wins', () => {
    const plan = planPoolSupremacy(
      parseTempHpPools([fakePoolRow(7, 8, 'campo-de-forca'), fakeDefenseRow(9)]),
      own,
      10,
    );
    expect(plan).toEqual({
      kind: 'apply',
      displaced: [{ effectId: 7, removed: true }],
      zeroWrites: [],
      deleteIds: [7],
    });
  });

  it('zeroes (never deletes) a smaller MIXED pool — its other modifiers live on', () => {
    const plan = planPoolSupremacy(parseTempHpPools([fakeMixedRow(5, 8)]), own, 10);
    expect(plan.kind).toBe('apply');
    if (plan.kind !== 'apply') return;
    expect(plan.displaced).toEqual([{ effectId: 5, removed: false }]);
    expect(plan.deleteIds).toEqual([]);
    const rewritten = JSON.parse(plan.zeroWrites[0]!.modifiers);
    expect(rewritten[0].amount).toBe(0);
    expect(rewritten[1].amount).toBe(4);
  });
});

describe('planDamage — temp-first routing, highest pool first', () => {
  it('pool covers all: hp untouched, partial drain persisted', () => {
    const plan = planDamage(parseTempHpPools([fakePoolRow(1, 10)]), 20, 7);
    expect(plan.hpCurrent).toBe(20);
    expect(plan.tempHpRemaining).toBe(3);
    expect(plan.drained).toEqual([{ effectId: 1, newAmount: 3, removed: false }]);
    expect(JSON.parse(plan.updates[0]!.modifiers)[0].amount).toBe(3);
  });

  it('drains multiple pools highest first, spilling the rest into hp', () => {
    const pools = parseTempHpPools([
      fakePoolRow(1, 4),
      fakePoolRow(2, 30, 'campo-de-forca'),
    ]);
    const plan = planDamage(pools, 20, 36);
    expect(plan.drained).toEqual([
      { effectId: 2, newAmount: 0, removed: true },
      { effectId: 1, newAmount: 0, removed: true },
    ]);
    expect(plan.deleteIds).toEqual([2, 1]);
    expect(plan.hpCurrent).toBe(18);
    expect(plan.tempHpRemaining).toBe(0);
  });

  it('passes straight through to hp when there is no pool', () => {
    const plan = planDamage([], 20, 5);
    expect(plan).toMatchObject({ hpCurrent: 15, tempHpRemaining: 0, drained: [] });
  });

  it('floors hp at 0 on overkill', () => {
    const plan = planDamage(parseTempHpPools([fakePoolRow(1, 2)]), 3, 99);
    expect(plan.hpCurrent).toBe(0);
    expect(plan.drained).toEqual([{ effectId: 1, newAmount: 0, removed: true }]);
  });

  it('zeroes an emptied MIXED pool instead of deleting the row', () => {
    const plan = planDamage(parseTempHpPools([fakeMixedRow(5, 6)]), 20, 6);
    expect(plan.drained).toEqual([{ effectId: 5, newAmount: 0, removed: false }]);
    expect(plan.deleteIds).toEqual([]);
    const rewritten = JSON.parse(plan.updates[0]!.modifiers);
    expect(rewritten[0].amount).toBe(0);
    expect(rewritten[1].amount).toBe(4);
  });
});
