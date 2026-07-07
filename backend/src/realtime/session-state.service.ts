import {
  BadRequestException,
  Injectable,
  Logger,
  NotFoundException,
} from '@nestjs/common';
import { randomUUID } from 'node:crypto';
import { z } from 'zod';
import { PrismaService } from '../prisma/prisma.service';

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

/**
 * Zod schema for the persisted blob. A corrupted or partial payload
 * fails loudly here rather than feeding malformed state to consumers
 * — the audit called out silent JSON.parse as an antipattern. Every
 * field has an explicit default so migration-produced empties parse
 * cleanly.
 */
const InitiativeEntrySchema = z.object({
  id: z.string(),
  label: z.string(),
  initiative: z.number(),
  type: z.enum(['character', 'npc']),
  characterId: z.number().optional(),
  hpCurrent: z.number().optional(),
  hpMax: z.number().optional(),
  mpCurrent: z.number().optional(),
  mpMax: z.number().optional(),
});
const SessionRuntimeStateSchema = z.object({
  initiative: z.array(InitiativeEntrySchema).default([]),
  round: z.number().default(0),
  turnIndex: z.number().default(-1),
});

/** Fresh empty-tracker factory. Returns a new mutable state each call
 * so different sessions don't accidentally share the initiative array. */
function emptyRuntimeState(): SessionRuntimeState {
  return { initiative: [], round: 0, turnIndex: -1 };
}

/** Hard ceiling on how many combatants can sit in one tracker. Two
 *  concerns: (1) a runaway "add" bug can't grow the blob unbounded,
 *  (2) the UI paginates poorly past ~20 rows anyway. 50 leaves plenty
 *  of headroom for large monster mob encounters. */
export const INITIATIVE_MAX_ENTRIES = 50;

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
  private readonly logger = new Logger(SessionStateService.name);
  private readonly states = new Map<number, SessionRuntimeState>();
  private readonly loading = new Map<number, Promise<SessionRuntimeState>>();
  /**
   * Sessions whose last persist failed. The next mutation on the same
   * session retries the write. Idempotent because `persist` always
   * serializes the current in-memory state — a stale flag can't drift
   * away from what the DB should hold.
   */
  private readonly dirty = new Set<number>();

  constructor(private readonly prisma: PrismaService) {}

  /**
   * Async accessor — hydrates from Session.runtimeState on the first
   * call so a restarted server sees the last-persisted tracker. The
   * mutating helpers below stay sync because callers already hold the
   * hydrated state (a socket that joined the room already awaited a
   * `getState` on connect).
   */
  async load(sessionId: number): Promise<SessionRuntimeState> {
    const cached = this.states.get(sessionId);
    if (cached) return cached;
    const inflight = this.loading.get(sessionId);
    if (inflight) return inflight;
    const promise = this.hydrate(sessionId).finally(() => {
      this.loading.delete(sessionId);
    });
    this.loading.set(sessionId, promise);
    return promise;
  }

  private async hydrate(sessionId: number): Promise<SessionRuntimeState> {
    let state: SessionRuntimeState = emptyRuntimeState();
    try {
      const row = await this.prisma.session.findUnique({
        where: { id: sessionId },
        select: { runtimeState: true },
      });
      if (row?.runtimeState) {
        state = this.parseBlob(row.runtimeState);
      }
    } catch (err) {
      this.logger.warn(
        `Session ${sessionId}: DB load failed, using empty state (${(err as Error).message})`,
      );
    }
    this.states.set(sessionId, state);
    return state;
  }

  private parseBlob(blob: string): SessionRuntimeState {
    try {
      const raw = JSON.parse(blob) as unknown;
      const parsed = SessionRuntimeStateSchema.safeParse(raw);
      if (parsed.success) return parsed.data;
      this.logger.warn(
        `Malformed runtimeState blob: ${parsed.error.issues[0]?.message ?? 'zod fail'}`,
      );
    } catch (err) {
      this.logger.warn(`runtimeState JSON parse failed: ${(err as Error).message}`);
    }
    return emptyRuntimeState();
  }

  getState(sessionId: number): SessionRuntimeState {
    return this.getOrCreate(sessionId);
  }

  addEntry(sessionId: number, input: AddEntryInput): SessionRuntimeState {
    const state = this.getOrCreate(sessionId);
    if (state.initiative.length >= INITIATIVE_MAX_ENTRIES) {
      throw new BadRequestException(
        `Initiative tracker is full (max ${INITIATIVE_MAX_ENTRIES} entries)`,
      );
    }
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

  /**
   * Refresh `hpMax` / `mpMax` on every entry that carries a
   * `characterId` — the DB Character row is the source of truth. Only
   * ceilings move; `hpCurrent`/`mpCurrent` are left untouched
   * (updateVitals is what changes them). Used by the gateway on
   * `get-session-state` so a level-up mid-session doesn't leave the
   * tracker stuck at the old max. Batched via a single findMany.
   */
  async refreshCharacterMaxes(sessionId: number): Promise<SessionRuntimeState> {
    const state = this.getOrCreate(sessionId);
    const charIds = Array.from(
      new Set(
        state.initiative
          .map((e) => e.characterId)
          .filter((id): id is number => typeof id === 'number'),
      ),
    );
    if (charIds.length === 0) return state;
    try {
      const rows = (await this.prisma.character.findMany({
        where: { id: { in: charIds } },
        select: { id: true, hpMax: true, mpMax: true },
      })) as { id: number; hpMax: number; mpMax: number }[];
      const byId = new Map(rows.map((r) => [r.id, r]));
      for (const entry of state.initiative) {
        if (entry.characterId === undefined) continue;
        const fresh = byId.get(entry.characterId);
        if (!fresh) continue;
        entry.hpMax = fresh.hpMax;
        entry.mpMax = fresh.mpMax;
      }
    } catch (err) {
      this.logger.warn(
        `Session ${sessionId} hpMax refresh failed: ${(err as Error).message}`,
      );
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
    this.dirty.delete(sessionId);
  }

  /**
   * Fire-and-forget serialize + write of the current in-memory state.
   * Returns a promise that never rejects; resolves to the new dirty
   * status so the gateway can broadcast a `persistence-warning` event
   * to connected clients when the flag flips.
   *
   * Callers (typically the RealtimeGateway) should call this after
   * every mutating handler + broadcast. Not awaiting keeps WS latency
   * low; the dirty-flag retry keeps the DB from drifting more than a
   * few mutations behind under transient failures.
   */
  persist(sessionId: number): Promise<boolean> {
    const state = this.states.get(sessionId);
    if (!state) return Promise.resolve(false);
    const blob = JSON.stringify(state);
    return this.prisma.session
      .update({
        where: { id: sessionId },
        data: { runtimeState: blob },
      })
      .then(() => {
        this.dirty.delete(sessionId);
        return false;
      })
      .catch((err: unknown) => {
        this.dirty.add(sessionId);
        this.logger.warn(
          `Session ${sessionId}: persist failed (${(err as Error).message}); marked dirty for retry`,
        );
        return true;
      });
  }

  /** True when the last persist attempt failed and hasn't been retried. */
  isDirty(sessionId: number): boolean {
    return this.dirty.has(sessionId);
  }

  /**
   * Set absolute hp/mp values on a specific entry. Clamps against the
   * max fields when present so a hpCurrent > hpMax broadcast can't
   * corrupt the tracker.
   *
   * When `WS_VITALS_WRITETHROUGH_LIVE=1` is set, the mutation also fires
   * a fire-and-forget Character.update so a page refresh mid-combat
   * sees the last committed state. Failures are logged and do not
   * block the WS response (session end still does a full batch commit
   * when `WS_VITALS_WRITETHROUGH=1` — see SessionsService.end).
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
    void this.maybeLiveWriteThrough(sessionId, next);
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
    void this.maybeLiveWriteThrough(sessionId, next);
    return state;
  }

  /**
   * Fire-and-forget Character.update mirroring the entry's current hp/mp
   * back to the DB row. Gated by `WS_VITALS_WRITETHROUGH_LIVE=1` so
   * production can opt in without a code change; default off keeps
   * historical behaviour (end-of-session commit only).
   *
   * Runs a fresh `findUnique` before the update to clamp against the
   * current DB `hpMax`/`mpMax` — the entry's cache can drift after a
   * level-up mid-session. Individual failures are `.warn` logged; we
   * do not surface them to the WS caller so a transient DB blip can't
   * kill an in-flight combat.
   */
  private async maybeLiveWriteThrough(
    sessionId: number,
    entry: InitiativeEntry,
  ): Promise<void> {
    if (process.env.WS_VITALS_WRITETHROUGH_LIVE !== '1') return;
    if (entry.characterId === undefined) return;
    if (entry.hpCurrent === undefined && entry.mpCurrent === undefined) {
      return;
    }
    try {
      const fresh = await this.prisma.character.findUnique({
        where: { id: entry.characterId },
        select: { hpMax: true, mpMax: true },
      });
      if (!fresh) return;
      const data: { hpCurrent?: number; mpCurrent?: number } = {};
      if (entry.hpCurrent !== undefined) {
        data.hpCurrent = Math.min(
          Math.max(0, entry.hpCurrent),
          fresh.hpMax,
        );
      }
      if (entry.mpCurrent !== undefined) {
        data.mpCurrent = Math.min(
          Math.max(0, entry.mpCurrent),
          fresh.mpMax,
        );
      }
      await this.prisma.character.update({
        where: { id: entry.characterId },
        data,
      });
    } catch (err) {
      this.logger.warn(
        `Session ${sessionId} live write-through failed for character ${entry.characterId}: ${(err as Error).message}`,
      );
    }
  }

  private getOrCreate(sessionId: number): SessionRuntimeState {
    let state = this.states.get(sessionId);
    if (!state) {
      /* First sync access without prior `load()`. Materializes an empty
       * tracker synchronously; the persisted state (if any) will hydrate
       * on the next async `load()`. Callers that need durability must
       * `await load()` before mutating. */
      state = emptyRuntimeState();
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
