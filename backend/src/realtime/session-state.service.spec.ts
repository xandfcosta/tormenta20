import { Test } from '@nestjs/testing';
import { NotFoundException } from '@nestjs/common';
import { SessionStateService } from './session-state.service';

/**
 * In-memory initiative state — pure logic, no dependencies.
 * Behaviour pinned:
 *   - fresh sessions materialize a default state
 *   - addEntry keeps the list sorted by initiative desc
 *   - updateEntry re-sorts and preserves the on-turn combatant
 *   - removeEntry adjusts turnIndex correctly (before / at / after)
 *   - nextTurn wraps + increments round
 *   - resetInitiative clears everything
 */

async function setup() {
  const module = await Test.createTestingModule({
    providers: [SessionStateService],
  }).compile();
  return { service: module.get(SessionStateService) };
}

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
