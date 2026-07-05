import { Injectable, NotFoundException } from '@nestjs/common';
import { randomUUID } from 'node:crypto';

export type InitiativeEntry = {
  id: string;
  label: string;
  initiative: number;
  type: 'character' | 'npc';
  characterId?: number;
  hpCurrent?: number;
  hpMax?: number;
  mpCurrent?: number;
  mpMax?: number;
};

export type SessionRuntimeState = {
  initiative: InitiativeEntry[];
  round: number;
  /**
   * Index into `initiative` (already sorted desc) pointing at the
   * combatant whose turn it currently is. `-1` before combat starts /
   * after a reset.
   */
  turnIndex: number;
};

export type AddEntryInput = Omit<InitiativeEntry, 'id'>;
export type UpdateEntryInput = Partial<Omit<InitiativeEntry, 'id'>>;

/**
 * In-memory per-session runtime state. Nothing is persisted — a server
 * restart wipes initiatives and turn counters. That's acceptable for
 * the MVP because sessions are short (~4h) and initiative resets each
 * combat scene anyway. Persistence can layer on later without changing
 * the service interface (just wire a repo behind `getOrCreate`).
 *
 * The initiative list is kept sorted DESC on every mutation so
 * consumers can render without a follow-up sort, and `turnIndex`
 * always points at the correct row even after inserts / deletes.
 */
@Injectable()
export class SessionStateService {
  private readonly states = new Map<number, SessionRuntimeState>();

  getState(sessionId: number): SessionRuntimeState {
    return this.getOrCreate(sessionId);
  }

  addEntry(sessionId: number, input: AddEntryInput): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    const currentTurn = state.initiative[state.turnIndex];
    const entry: InitiativeEntry = { id: randomUUID(), ...input };
    state.initiative.push(entry);
    this.sortInitiative(state);
    /* Preserve who's currently on turn when the sort shuffles indices. */
    if (currentTurn) {
      state.turnIndex = state.initiative.findIndex((e) => e.id === currentTurn.id);
    }
    return state;
  }

  updateEntry(
    sessionId: number,
    entryId: string,
    patch: UpdateEntryInput,
  ): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    const idx = state.initiative.findIndex((e) => e.id === entryId);
    if (idx < 0) {
      throw new NotFoundException(`Entry ${entryId} not found`);
    }
    state.initiative[idx] = { ...state.initiative[idx]!, ...patch };
    if (patch.initiative !== undefined) {
      const currentTurn = state.initiative[state.turnIndex];
      this.sortInitiative(state);
      if (currentTurn) {
        state.turnIndex = state.initiative.findIndex(
          (e) => e.id === currentTurn.id,
        );
      }
    }
    return state;
  }

  removeEntry(sessionId: number, entryId: string): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    const idx = state.initiative.findIndex((e) => e.id === entryId);
    if (idx < 0) {
      throw new NotFoundException(`Entry ${entryId} not found`);
    }
    state.initiative.splice(idx, 1);
    if (state.initiative.length === 0) {
      state.turnIndex = -1;
      return state;
    }
    if (idx < state.turnIndex) {
      state.turnIndex -= 1;
    } else if (idx === state.turnIndex) {
      /* Removed the row on turn — the next combatant slides into place;
       * turnIndex stays the same but may now point past the tail. */
      if (state.turnIndex >= state.initiative.length) {
        state.turnIndex = 0;
        state.round += 1;
      }
    }
    return state;
  }

  /**
   * Advance to the next combatant. Wraps around to index 0 and
   * increments `round`. Starting from the pre-combat state (turnIndex
   * = -1) puts the first combatant on turn without bumping the round.
   */
  nextTurn(sessionId: number): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    if (state.initiative.length === 0) return state;
    if (state.turnIndex < 0) {
      state.turnIndex = 0;
      if (state.round < 1) state.round = 1;
      return state;
    }
    state.turnIndex += 1;
    if (state.turnIndex >= state.initiative.length) {
      state.turnIndex = 0;
      state.round += 1;
    }
    return state;
  }

  /** Clear initiative + round counter. Keeps the session tracked. */
  resetInitiative(sessionId: number): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    state.initiative = [];
    state.round = 0;
    state.turnIndex = -1;
    return state;
  }

  /** Drop the session entirely — useful when the DB row is deleted. */
  forget(sessionId: number): void {
    this.states.delete(sessionId);
  }

  /**
   * Set absolute hp/mp values on a specific entry. Clamps against the
   * max fields when present so a hpCurrent > hpMax broadcast can't
   * corrupt the tracker.
   *
   * TODO(persistence): when `entry.characterId` is set, downstream we
   * likely want to persist hpCurrent/mpCurrent back to the DB Character
   * row so a page refresh doesn't lose the state. Kept out of scope
   * for the MVP realtime loop; add a repo dependency + write-through
   * when the product needs it.
   */
  patchVitals(
    sessionId: number,
    entryId: string,
    patch: { hpCurrent?: number; mpCurrent?: number },
  ): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    const idx = state.initiative.findIndex((e) => e.id === entryId);
    if (idx < 0) {
      throw new NotFoundException(`Entry ${entryId} not found`);
    }
    const entry = state.initiative[idx]!;
    const next: InitiativeEntry = { ...entry };
    if (patch.hpCurrent !== undefined) {
      next.hpCurrent = clampVital(patch.hpCurrent, entry.hpMax);
    }
    if (patch.mpCurrent !== undefined) {
      next.mpCurrent = clampVital(patch.mpCurrent, entry.mpMax);
    }
    state.initiative[idx] = next;
    return state;
  }

  /**
   * Apply a hp/mp delta ("sofreu 10 de dano" → hpDelta: -10). Absent
   * hpCurrent counts as 0. Missing max is treated as no upper bound.
   */
  deltaVitals(
    sessionId: number,
    entryId: string,
    patch: { hpDelta?: number; mpDelta?: number },
  ): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    const idx = state.initiative.findIndex((e) => e.id === entryId);
    if (idx < 0) {
      throw new NotFoundException(`Entry ${entryId} not found`);
    }
    const entry = state.initiative[idx]!;
    const next: InitiativeEntry = { ...entry };
    if (patch.hpDelta !== undefined) {
      const raw = (entry.hpCurrent ?? 0) + patch.hpDelta;
      next.hpCurrent = clampVital(raw, entry.hpMax);
    }
    if (patch.mpDelta !== undefined) {
      const raw = (entry.mpCurrent ?? 0) + patch.mpDelta;
      next.mpCurrent = clampVital(raw, entry.mpMax);
    }
    state.initiative[idx] = next;
    return state;
  }

  private getOrCreate(sessionId: number): SessionRuntimeState {
    let state = this.states.get(sessionId);
    if (!state) {
      state = { initiative: [], round: 0, turnIndex: -1 };
      this.states.set(sessionId, state);
    }
    return state;
  }

  private sortInitiative(state: SessionRuntimeState) {
    state.initiative.sort(
      (a, b) => b.initiative - a.initiative || a.label.localeCompare(b.label),
    );
  }
}

/**
 * Clamp a vital value against an optional max. Negative floor is 0
 * (T20 conventions treat "morrer" as reaching -PVmáx, but the tracker
 * itself never goes below 0 for display purposes — a character on 0
 * is "unconscious/dying" and the GM handles the rest narratively).
 */
function clampVital(value: number, max: number | undefined): number {
  const floored = Math.max(0, value);
  if (max === undefined) return floored;
  return Math.min(floored, max);
}
