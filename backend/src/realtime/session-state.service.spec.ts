jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { Test } from '@nestjs/testing';
import { ConfigService } from '@nestjs/config';
import { NotFoundException } from '@nestjs/common';
import { SessionStateService } from './session-state.service';
import { PrismaService } from '../prisma/prisma.service';

/**
 * In-memory initiative state — the P1a persistence layer sits on top
 * of pure logic that these specs already exercise. To keep the
 * pre-P1a specs green, we inject a fake PrismaService whose
 * `session.findUnique` returns `null` (behaves like the pre-P1a "no
 * DB" world). Persistence-specific specs live below with an
 * override that returns a runtimeState blob.
 */

const emptySessionRepo = { findUnique: async () => null };

async function setup(over?: {
  sessionFindUnique?: (args: unknown) => Promise<unknown>;
  sessionUpdate?: jest.Mock;
  characterFindMany?: jest.Mock;
  characterFindUnique?: jest.Mock;
  characterUpdate?: jest.Mock;
}) {
  const sessionUpdate =
    over?.sessionUpdate ?? jest.fn().mockResolvedValue({});
  const characterFindMany =
    over?.characterFindMany ?? jest.fn().mockResolvedValue([]);
  const characterFindUnique =
    over?.characterFindUnique ??
    jest.fn().mockResolvedValue({ hpMax: 100, mpMax: 20 });
  const characterUpdate =
    over?.characterUpdate ?? jest.fn().mockResolvedValue({});
  const module = await Test.createTestingModule({
    providers: [
      SessionStateService,
      { provide: ConfigService, useValue: { get: (k: string) => process.env[k] } },
      {
        provide: PrismaService,
        useValue: {
          session: {
            findUnique: over?.sessionFindUnique ?? emptySessionRepo.findUnique,
            update: sessionUpdate,
          },
          /* P5: refreshCharacterMaxes reads here. Empty default so
           * pre-P5 specs stay green (nothing gets updated). Live
           * write-through reads via findUnique + update. */
          character: {
            findMany: characterFindMany,
            findUnique: characterFindUnique,
            update: characterUpdate,
          },
        },
      },
    ],
  }).compile();
  return {
    service: module.get(SessionStateService),
    sessionUpdate,
    characterFindMany,
    characterFindUnique,
    characterUpdate,
  };
}

describe('SessionStateService.load (P1a persistence)', () => {
  it('hydrates from the persisted runtimeState blob', async () => {
    const payload = JSON.stringify({
      initiative: [
        {
          id: 'x',
          label: 'Goblin',
          initiative: 12,
          type: 'npc',
          hpCurrent: 4,
          hpMax: 4,
        },
      ],
      round: 2,
      turnIndex: 0,
    });
    const { service } = await setup({
      sessionFindUnique: async () => ({ runtimeState: payload }),
    });
    const state = await service.load(1);
    expect(state.round).toBe(2);
    expect(state.turnIndex).toBe(0);
    expect(state.initiative).toHaveLength(1);
    expect(state.initiative[0]!.label).toBe('Goblin');
  });

  it('falls back to empty state on malformed JSON', async () => {
    const { service } = await setup({
      sessionFindUnique: async () => ({ runtimeState: 'not-json{' }),
    });
    const state = await service.load(1);
    expect(state).toEqual({ initiative: [], round: 0, turnIndex: -1 });
  });

  it('falls back to empty state on schema mismatch', async () => {
    const { service } = await setup({
      sessionFindUnique: async () =>
        ({ runtimeState: JSON.stringify({ round: 'wrong-type' }) }),
    });
    const state = await service.load(1);
    expect(state).toEqual({ initiative: [], round: 0, turnIndex: -1 });
  });

  it('falls back to empty state when Prisma throws', async () => {
    const { service } = await setup({
      sessionFindUnique: async () => {
        throw new Error('DB down');
      },
    });
    const state = await service.load(1);
    expect(state).toEqual({ initiative: [], round: 0, turnIndex: -1 });
  });

  it('returns the cached state on the second load without a second DB hit', async () => {
    const findUnique = jest.fn(async () => ({
      runtimeState: JSON.stringify({ initiative: [], round: 5, turnIndex: -1 }),
    }));
    const { service } = await setup({ sessionFindUnique: findUnique });
    const first = await service.load(1);
    const second = await service.load(1);
    expect(first).toBe(second);
    expect(findUnique).toHaveBeenCalledTimes(1);
  });

  it('de-dupes concurrent loads (single DB hit under contention)', async () => {
    let calls = 0;
    const findUnique = jest.fn(async () => {
      calls++;
      await new Promise((r) => setTimeout(r, 10));
      return { runtimeState: JSON.stringify({ round: 3 }) };
    });
    const { service } = await setup({ sessionFindUnique: findUnique });
    const [a, b, c] = await Promise.all([
      service.load(1),
      service.load(1),
      service.load(1),
    ]);
    expect(a).toBe(b);
    expect(b).toBe(c);
    expect(calls).toBe(1);
  });
});

describe('SessionStateService.persist (P1b fire-and-forget)', () => {
  it('writes the current state to Session.runtimeState', async () => {
    const { service, sessionUpdate } = await setup();
    service.addEntry(1, { label: 'A', initiative: 12, type: 'npc' });
    await service.persist(1);
    expect(sessionUpdate).toHaveBeenCalledWith(
      expect.objectContaining({
        where: { id: 1 },
        data: expect.objectContaining({
          runtimeState: expect.any(String),
        }),
      }),
    );
    const call = sessionUpdate.mock.calls.at(-1)![0]!;
    const parsed = JSON.parse(call.data.runtimeState);
    expect(parsed.initiative[0]!.label).toBe('A');
  });

  it('never rejects — DB failure marks the session dirty', async () => {
    const sessionUpdate = jest.fn().mockRejectedValue(new Error('DB down'));
    const { service } = await setup({ sessionUpdate });
    service.addEntry(1, { label: 'A', initiative: 12, type: 'npc' });
    /* If the promise rejected, this test would throw. */
    await service.persist(1);
    expect(service.isDirty(1)).toBe(true);
  });

  it('clears the dirty flag on a successful retry', async () => {
    const sessionUpdate = jest
      .fn()
      .mockRejectedValueOnce(new Error('DB down'))
      .mockResolvedValue({});
    const { service } = await setup({ sessionUpdate });
    service.addEntry(1, { label: 'A', initiative: 12, type: 'npc' });
    await service.persist(1);
    expect(service.isDirty(1)).toBe(true);
    await service.persist(1);
    expect(service.isDirty(1)).toBe(false);
    expect(sessionUpdate).toHaveBeenCalledTimes(2);
  });

  it('returns the new dirty status (true on failure, false on success)', async () => {
    /* Persistence-warning gate: the gateway broadcasts based on this
     * boolean. `persist` must resolve to `true` when the write failed
     * (dirty just set) and `false` when it succeeded (dirty just
     * cleared). */
    const sessionUpdate = jest
      .fn()
      .mockRejectedValueOnce(new Error('DB down'))
      .mockResolvedValue({});
    const { service } = await setup({ sessionUpdate });
    service.addEntry(1, { label: 'A', initiative: 12, type: 'npc' });
    await expect(service.persist(1)).resolves.toBe(true);
    await expect(service.persist(1)).resolves.toBe(false);
  });

  it('no-ops when the session was never touched (nothing in memory)', async () => {
    const { service, sessionUpdate } = await setup();
    await service.persist(42);
    expect(sessionUpdate).not.toHaveBeenCalled();
  });

  it('forget clears both cache and dirty flag', async () => {
    const sessionUpdate = jest.fn().mockRejectedValue(new Error('DB down'));
    const { service } = await setup({ sessionUpdate });
    service.addEntry(1, { label: 'A', initiative: 12, type: 'npc' });
    await service.persist(1);
    expect(service.isDirty(1)).toBe(true);
    service.forget(1);
    expect(service.isDirty(1)).toBe(false);
  });
});

describe('SessionStateService.getState', () => {
  it('returns a default fresh state for a new session', async () => {
    const { service } = await setup();
    const state = service.getState(42);
    expect(state.initiative).toEqual([]);
    expect(state.round).toBe(0);
    expect(state.turnIndex).toBe(-1);
  });

  it('is idempotent across calls (same reference)', async () => {
    const { service } = await setup();
    const a = service.getState(1);
    const b = service.getState(1);
    expect(a).toBe(b);
  });
});

describe('SessionStateService P5 caps + hpMax refresh', () => {
  it('addEntry throws BadRequest when tracker already at 50 entries', async () => {
    const { service } = await setup();
    for (let i = 0; i < 50; i++) {
      service.addEntry(1, {
        label: `NPC${i}`,
        initiative: 10 - (i % 5),
        type: 'npc',
      });
    }
    expect(() =>
      service.addEntry(1, { label: 'Overflow', initiative: 1, type: 'npc' }),
    ).toThrow(/Initiative tracker is full/);
  });

  it('refreshCharacterMaxes updates hpMax/mpMax from DB without touching hpCurrent', async () => {
    const characterFindMany = jest.fn().mockResolvedValue([
      { id: 10, hpMax: 25, mpMax: 8 },
    ]);
    const { service } = await setup({ characterFindMany });
    service.addEntry(1, {
      label: 'PC',
      initiative: 12,
      type: 'character',
      characterId: 10,
      hpCurrent: 7,
      hpMax: 15,
      mpCurrent: 2,
      mpMax: 4,
    });
    const state = await service.refreshCharacterMaxes(1);
    const entry = state.initiative[0]!;
    expect(entry.hpMax).toBe(25);
    expect(entry.mpMax).toBe(8);
    expect(entry.hpCurrent).toBe(7);
    expect(entry.mpCurrent).toBe(2);
  });

  it('refreshCharacterMaxes skips entries without characterId', async () => {
    const characterFindMany = jest.fn().mockResolvedValue([]);
    const { service } = await setup({ characterFindMany });
    service.addEntry(1, {
      label: 'Goblin',
      initiative: 8,
      type: 'npc',
      hpCurrent: 3,
      hpMax: 10,
    });
    await service.refreshCharacterMaxes(1);
    expect(characterFindMany).not.toHaveBeenCalled();
  });

  it('refreshCharacterMaxes tolerates DB failure — returns state unchanged', async () => {
    const characterFindMany = jest
      .fn()
      .mockRejectedValue(new Error('DB down'));
    const { service } = await setup({ characterFindMany });
    service.addEntry(1, {
      label: 'PC',
      initiative: 12,
      type: 'character',
      characterId: 10,
      hpMax: 15,
    });
    const state = await service.refreshCharacterMaxes(1);
    expect(state.initiative[0]!.hpMax).toBe(15);
  });
});

describe('SessionStateService.addEntry', () => {
  it('appends and sorts by initiative desc', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'Slow', initiative: 5, type: 'character' });
    service.addEntry(1, { label: 'Fast', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'Mid', initiative: 12, type: 'character' });
    const labels = service.getState(1).initiative.map((e) => e.label);
    expect(labels).toEqual(['Fast', 'Mid', 'Slow']);
  });

  it('breaks ties by label alphabetically', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'B', initiative: 10, type: 'npc' });
    service.addEntry(1, { label: 'A', initiative: 10, type: 'npc' });
    expect(service.getState(1).initiative.map((e) => e.label)).toEqual([
      'A',
      'B',
    ]);
  });

  it('preserves who is on turn after a re-sort', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 15, type: 'npc' });
    service.nextTurn(1); // A on turn
    service.nextTurn(1); // B on turn
    service.addEntry(1, { label: 'C', initiative: 25, type: 'npc' });
    const state = service.getState(1);
    expect(state.initiative[state.turnIndex]?.label).toBe('B');
  });
});

describe('SessionStateService.updateEntry', () => {
  it('applies the patch to a specific entry', async () => {
    const { service } = await setup();
    const s1 = service.addEntry(1, {
      label: 'Goblin',
      initiative: 8,
      type: 'npc',
      hpCurrent: 10,
      hpMax: 10,
    });
    const id = s1.initiative[0]!.id;
    service.updateEntry(1, id, { hpCurrent: 3 });
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(3);
  });

  it('re-sorts when initiative changes', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 10, type: 'npc' });
    const bId = service.getState(1).initiative.find((e) => e.label === 'B')!.id;
    service.updateEntry(1, bId, { initiative: 30 });
    expect(service.getState(1).initiative.map((e) => e.label)).toEqual([
      'B',
      'A',
    ]);
  });

  it('throws NotFound for unknown entryId', async () => {
    const { service } = await setup();
    expect(() =>
      service.updateEntry(1, 'ghost', { label: 'z' }),
    ).toThrow(NotFoundException);
  });
});

describe('SessionStateService.removeEntry', () => {
  it('drops the row and clears the turn when empty', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'Lonely',
      initiative: 10,
      type: 'npc',
    });
    service.nextTurn(1);
    const id = s.initiative[0]!.id;
    service.removeEntry(1, id);
    const state = service.getState(1);
    expect(state.initiative).toEqual([]);
    expect(state.turnIndex).toBe(-1);
  });

  it('decrements turnIndex when a row before the turn is removed', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 15, type: 'npc' });
    service.addEntry(1, { label: 'C', initiative: 10, type: 'npc' });
    service.nextTurn(1); // A
    service.nextTurn(1); // B
    const aId = service.getState(1).initiative[0]!.id;
    service.removeEntry(1, aId);
    const state = service.getState(1);
    expect(state.initiative[state.turnIndex]?.label).toBe('B');
    expect(state.turnIndex).toBe(0);
  });

  it('advances the round when the last combatant on turn is removed', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 15, type: 'npc' });
    service.nextTurn(1); // A
    service.nextTurn(1); // B (last)
    const bId = service.getState(1).initiative[1]!.id;
    service.removeEntry(1, bId);
    const state = service.getState(1);
    expect(state.initiative[state.turnIndex]?.label).toBe('A');
    expect(state.round).toBe(2);
  });

  it('throws NotFound for unknown entryId', async () => {
    const { service } = await setup();
    expect(() => service.removeEntry(1, 'ghost')).toThrow(NotFoundException);
  });
});

describe('SessionStateService.nextTurn', () => {
  it('starts combat with turnIndex 0 and round 1', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 10, type: 'npc' });
    const state = service.nextTurn(1);
    expect(state.turnIndex).toBe(0);
    expect(state.round).toBe(1);
  });

  it('wraps to 0 and increments round after the last combatant', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.addEntry(1, { label: 'B', initiative: 10, type: 'npc' });
    service.nextTurn(1); // A, round 1
    service.nextTurn(1); // B, round 1
    const state = service.nextTurn(1); // wrap → A, round 2
    expect(state.turnIndex).toBe(0);
    expect(state.round).toBe(2);
  });

  it('no-op on an empty initiative list', async () => {
    const { service } = await setup();
    const state = service.nextTurn(1);
    expect(state.turnIndex).toBe(-1);
    expect(state.round).toBe(0);
  });
});

describe('SessionStateService.patchVitals', () => {
  it('applies absolute values clamped by hpMax/mpMax', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 20,
      hpMax: 30,
      mpCurrent: 5,
      mpMax: 10,
    });
    const id = s.initiative[0]!.id;
    /* hpCurrent above max should clamp to max */
    service.patchVitals(1, id, { hpCurrent: 100 });
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(30);
    /* mpCurrent below 0 should clamp to 0 */
    service.patchVitals(1, id, { mpCurrent: -5 });
    expect(service.getState(1).initiative[0]?.mpCurrent).toBe(0);
  });

  it('leaves untouched fields alone', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 20,
      hpMax: 30,
      mpCurrent: 5,
      mpMax: 10,
    });
    const id = s.initiative[0]!.id;
    service.patchVitals(1, id, { hpCurrent: 10 });
    const entry = service.getState(1).initiative[0]!;
    expect(entry.hpCurrent).toBe(10);
    expect(entry.mpCurrent).toBe(5);
  });

  it('throws NotFound for unknown entryId', async () => {
    const { service } = await setup();
    expect(() => service.patchVitals(1, 'ghost', { hpCurrent: 10 })).toThrow(
      NotFoundException,
    );
  });
});

describe('SessionStateService.deltaVitals', () => {
  it('applies hpDelta and clamps to max', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 25,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { hpDelta: 10 });
    /* 25 + 10 = 35 → clamped to 30 */
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(30);
  });

  it('negative delta ("suffered damage") clamps at 0', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      hpCurrent: 5,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { hpDelta: -20 });
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(0);
  });
});

describe('SessionStateService — WS_VITALS_WRITETHROUGH_LIVE', () => {
  const OLD = process.env.WS_VITALS_WRITETHROUGH_LIVE;
  afterEach(() => {
    if (OLD === undefined) delete process.env.WS_VITALS_WRITETHROUGH_LIVE;
    else process.env.WS_VITALS_WRITETHROUGH_LIVE = OLD;
    jest.clearAllMocks();
  });

  /**
   * Small helper: patchVitals / deltaVitals fire the write-through as
   * `void this.maybeLiveWriteThrough(...)` so the mock resolution needs
   * a microtask flush before the assertion. Using a promise chain we
   * fully control instead of an arbitrary setTimeout keeps the test
   * deterministic.
   */
  const flush = () => Promise.resolve().then(() => Promise.resolve());

  it('flag off: no character.update fires on patchVitals', async () => {
    delete process.env.WS_VITALS_WRITETHROUGH_LIVE;
    const { service, characterUpdate } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      characterId: 42,
      hpCurrent: 20,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    service.patchVitals(1, id, { hpCurrent: 15 });
    await flush();
    expect(characterUpdate).not.toHaveBeenCalled();
  });

  it('flag on: patchVitals mirrors hp to Character row (clamped to fresh hpMax)', async () => {
    process.env.WS_VITALS_WRITETHROUGH_LIVE = '1';
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ hpMax: 25, mpMax: 10 });
    const characterUpdate = jest.fn().mockResolvedValue({});
    const { service } = await setup({ characterFindUnique, characterUpdate });
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      characterId: 42,
      hpCurrent: 20,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    service.patchVitals(1, id, { hpCurrent: 40 });
    await flush();
    // clamped by entry.hpMax=30 first (patchVitals), then fresh hpMax=25
    // in write-through.
    expect(characterUpdate).toHaveBeenCalledWith({
      where: { id: 42 },
      data: { hpCurrent: 25 },
    });
  });

  it('flag on: skips NPC entries (no characterId)', async () => {
    process.env.WS_VITALS_WRITETHROUGH_LIVE = '1';
    const { service, characterUpdate } = await setup();
    const s = service.addEntry(1, {
      label: 'Goblin',
      initiative: 15,
      type: 'npc',
      hpCurrent: 5,
      hpMax: 10,
    });
    const id = s.initiative[0]!.id;
    service.patchVitals(1, id, { hpCurrent: 3 });
    await flush();
    expect(characterUpdate).not.toHaveBeenCalled();
  });

  it('flag on: deltaVitals also fires the mirror', async () => {
    process.env.WS_VITALS_WRITETHROUGH_LIVE = '1';
    const characterFindUnique = jest
      .fn()
      .mockResolvedValue({ hpMax: 30, mpMax: 10 });
    const characterUpdate = jest.fn().mockResolvedValue({});
    const { service } = await setup({ characterFindUnique, characterUpdate });
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      characterId: 7,
      hpCurrent: 20,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { hpDelta: -5 });
    await flush();
    expect(characterUpdate).toHaveBeenCalledWith({
      where: { id: 7 },
      data: { hpCurrent: 15 },
    });
  });

  it('flag on: DB failure logged, does not throw', async () => {
    process.env.WS_VITALS_WRITETHROUGH_LIVE = '1';
    const characterUpdate = jest.fn().mockRejectedValue(new Error('boom'));
    const { service } = await setup({ characterUpdate });
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      characterId: 42,
      hpCurrent: 20,
      hpMax: 30,
    });
    const id = s.initiative[0]!.id;
    expect(() =>
      service.patchVitals(1, id, { hpCurrent: 15 }),
    ).not.toThrow();
    await flush();
  });

  it('missing hpCurrent counts as 0', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'NPC',
      initiative: 10,
      type: 'npc',
      hpMax: 20,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { hpDelta: 5 });
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(5);
  });

  it('no max = no upper bound (only floor 0)', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'Boss',
      initiative: 25,
      type: 'npc',
      hpCurrent: 100,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { hpDelta: 500 });
    expect(service.getState(1).initiative[0]?.hpCurrent).toBe(600);
  });

  it('applies mpDelta the same way', async () => {
    const { service } = await setup();
    const s = service.addEntry(1, {
      label: 'PC',
      initiative: 15,
      type: 'character',
      mpCurrent: 3,
      mpMax: 8,
    });
    const id = s.initiative[0]!.id;
    service.deltaVitals(1, id, { mpDelta: -2 });
    expect(service.getState(1).initiative[0]?.mpCurrent).toBe(1);
  });

  it('throws NotFound for unknown entryId', async () => {
    const { service } = await setup();
    expect(() => service.deltaVitals(1, 'ghost', { hpDelta: 1 })).toThrow(
      NotFoundException,
    );
  });
});

describe('SessionStateService.resetInitiative', () => {
  it('clears everything but keeps the session tracked', async () => {
    const { service } = await setup();
    service.addEntry(1, { label: 'A', initiative: 20, type: 'npc' });
    service.nextTurn(1);
    service.resetInitiative(1);
    const state = service.getState(1);
    expect(state.initiative).toEqual([]);
    expect(state.round).toBe(0);
    expect(state.turnIndex).toBe(-1);
  });
});
