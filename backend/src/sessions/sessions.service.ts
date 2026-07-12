import {
  BadRequestException,
  Inject,
  Injectable,
  Logger,
  NotFoundException,
  Optional,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CampaignsService } from '../campaigns/campaigns.service';
import { CreateSessionDto, UpdateSessionDto } from './dto/session.dto';
import type { SessionStateService } from '../realtime/session-state.service';

export type SessionStatus = 'planned' | 'active' | 'ended';

/** DI token for SessionStateService — string-based so unit tests can
 * substitute a fake without importing the real class (which pulls in
 * Prisma runtime). RealtimeModule provides the real service under the
 * same token. */
export const SESSION_STATE_SERVICE = 'SESSION_STATE_SERVICE' as const;

@Injectable()
export class SessionsService {
  private readonly logger = new Logger(SessionsService.name);

  constructor(
    private readonly prisma: PrismaService,
    private readonly campaigns: CampaignsService,
    /* SessionStateService is a runtime-only concern; keep the
     * dependency `Optional` so unit tests that don't care about the
     * tracker can compile without wiring it. */
    @Optional()
    @Inject(SESSION_STATE_SERVICE)
    private readonly state?: SessionStateService,
  ) {}

  /**
   * Owner-only guard for **write** paths (create / update / delete /
   * start / end / clearTracker). Session mutations stay GM-exclusive.
   */
  private async assertCampaignOwnership(ownerId: number, campaignId: number) {
    await this.campaigns.findOne(ownerId, campaignId);
  }

  /**
   * Member-aware guard for **read** paths — GM or any player member.
   * Returns the caller's role so callers (WS gateway) can gate
   * finer-grained actions later.
   */
  private async assertCampaignAccess(userId: number, campaignId: number) {
    const { role } = await this.campaigns.resolveAccess(userId, campaignId);
    return role;
  }

  /** Member-aware session list — GM or player member. */
  async listForCaller(userId: number, campaignId: number) {
    await this.assertCampaignAccess(userId, campaignId);
    return this.prisma.session.findMany({
      where: { campaignId },
      orderBy: { sessionNumber: 'asc' },
    });
  }

  /** Owner-only session lookup — used by every write path below. */
  async findOne(ownerId: number, campaignId: number, id: number) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    return this.loadSession(campaignId, id);
  }

  /**
   * Member-aware session lookup — GM or player member. Returns the
   * session plus caller role so the realtime gateway can stash it for
   * per-action gating (e.g. GM-only turn advance).
   */
  async findOneForCaller(userId: number, campaignId: number, id: number) {
    const role = await this.assertCampaignAccess(userId, campaignId);
    const session = await this.loadSession(campaignId, id);
    return { session, role };
  }

  private async loadSession(campaignId: number, id: number) {
    const session = await this.prisma.session.findUnique({ where: { id } });
    if (!session || session.campaignId !== campaignId) {
      throw new NotFoundException(`Session ${id} not found`);
    }
    return session;
  }

  async create(
    ownerId: number,
    campaignId: number,
    dto: CreateSessionDto,
  ) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    const title = dto.title?.trim() || null;
    const notes = dto.notes?.trim() || null;
    return this.prisma.session.create({
      data: {
        campaignId,
        sessionNumber: dto.sessionNumber,
        title,
        notes,
      },
    });
  }

  async update(
    ownerId: number,
    campaignId: number,
    id: number,
    dto: UpdateSessionDto,
  ) {
    await this.findOne(ownerId, campaignId, id);
    const data: {
      sessionNumber?: number;
      title?: string | null;
      notes?: string | null;
    } = {};
    if (dto.sessionNumber !== undefined) data.sessionNumber = dto.sessionNumber;
    if (dto.title !== undefined) data.title = dto.title.trim() || null;
    if (dto.notes !== undefined) data.notes = dto.notes.trim() || null;
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.session.update({ where: { id }, data });
  }

  async remove(ownerId: number, campaignId: number, id: number) {
    await this.findOne(ownerId, campaignId, id);
    await this.prisma.session.delete({ where: { id } });
    return { id };
  }

  /**
   * Transition to `active`. Allowed from any prior state:
   *   - planned → active (first start; sets startedAt)
   *   - active  → no-op (idempotent)
   *   - ended   → active (reopen; keeps the prior startedAt, clears
   *     endedAt so the UI can render "still ongoing"). Reopening is
   *     the right default UX because GMs pause combat mid-scene all
   *     the time (dinner break, TPK cliffhanger) — forcing a "create
   *     new session" ritual just to keep initiative alive is friction.
   *     The runtime tracker is preserved across the transition.
   */
  async start(ownerId: number, campaignId: number, id: number) {
    const session = await this.findOne(ownerId, campaignId, id);
    if (session.status === 'active') return session;
    if (session.status === 'ended') {
      this.logger.log(`Session ${id} reopened (was ended)`);
      return this.prisma.session.update({
        where: { id },
        data: { status: 'active', endedAt: null },
      });
    }
    return this.prisma.session.update({
      where: { id },
      data: { status: 'active', startedAt: new Date() },
    });
  }

  /**
   * Transition active → ended. Rejects on planned (nothing to end) and
   * idempotent on ended (returns the current row). Runtime state is
   * intentionally preserved — reopen or `clearTracker` are the two
   * explicit knobs.
   *
   * P3 batch commit: when `WS_VITALS_WRITETHROUGH=1` and the tracker
   * has entries linked to Character rows (via `characterId`), the end
   * transition also flushes each entry's `hpCurrent`/`mpCurrent` back
   * to the DB Character row. Clamped against the fresh `Character.hpMax`
   * (source of truth — the entry cache can be stale after a level-up
   * mid-session). Skipped by default because the WS layer already
   * mutates in-memory; opting in shouldn't be automatic until a
   * product decision to sync sheets happens.
   */
  async end(ownerId: number, campaignId: number, id: number) {
    const session = await this.findOne(ownerId, campaignId, id);
    if (session.status === 'planned') {
      throw new BadRequestException(
        `Session ${id} was never started; nothing to end`,
      );
    }
    if (session.status === 'ended') return session;
    if (this.state && process.env.WS_VITALS_WRITETHROUGH === '1') {
      await this.commitVitalsToCharacters(id);
    }
    return this.prisma.session.update({
      where: { id },
      data: { status: 'ended', endedAt: new Date() },
    });
  }

  /**
   * Walk the runtime initiative list; for every entry that carries a
   * `characterId`, clamp its hp/mp to the fresh `Character.hpMax` /
   * `mpMax` and write. Individual failures are logged and skipped —
   * one bad row shouldn't block the whole batch.
   */
  private async commitVitalsToCharacters(sessionId: number) {
    if (!this.state) return;
    const runtime = this.state.getState(sessionId);
    for (const entry of runtime.initiative) {
      if (entry.characterId === undefined) continue;
      if (
        entry.hpCurrent === undefined &&
        entry.mpCurrent === undefined
      ) {
        continue;
      }
      try {
        const fresh = await this.prisma.character.findUnique({
          where: { id: entry.characterId },
          select: { hpMax: true, mpMax: true },
        });
        if (!fresh) continue;
        const patch: { hpCurrent?: number; mpCurrent?: number } = {};
        if (entry.hpCurrent !== undefined) {
          patch.hpCurrent = Math.min(
            Math.max(0, entry.hpCurrent),
            fresh.hpMax,
          );
        }
        if (entry.mpCurrent !== undefined) {
          patch.mpCurrent = Math.min(
            Math.max(0, entry.mpCurrent),
            fresh.mpMax,
          );
        }
        await this.prisma.character.update({
          where: { id: entry.characterId },
          data: patch,
        });
      } catch (err) {
        this.logger.warn(
          `Session ${sessionId} vitals commit failed for character ${entry.characterId}: ${(err as Error).message}`,
        );
      }
    }
  }

  /**
   * Wipe the initiative tracker for a session without touching its
   * lifecycle state. Splits "clear combat" (frequent — end of a scene,
   * new encounter) from "end the session" (rare — GM is done for the
   * day). Persists the empty tracker so a page refresh doesn't
   * resurrect the pre-clear list.
   */
  async clearTracker(ownerId: number, campaignId: number, id: number) {
    await this.findOne(ownerId, campaignId, id);
    if (!this.state) return { id };
    this.state.resetInitiative(id);
    await this.state.persist(id);
    return { id };
  }
}
