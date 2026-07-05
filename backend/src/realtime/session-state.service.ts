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
