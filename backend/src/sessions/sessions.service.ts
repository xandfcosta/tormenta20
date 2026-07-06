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
   * All read/write paths run `campaigns.findOne(ownerId, campaignId)`
   * first — that method already throws NotFound / Forbidden, so ownership
   * enforcement lives in one place instead of being duplicated per verb.
   */
  private async assertCampaignOwnership(ownerId: number, campaignId: number) {
    await this.campaigns.findOne(ownerId, campaignId);
  }

  async list(ownerId: number, campaignId: number) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    return this.prisma.session.findMany({
      where: { campaignId },
      orderBy: { sessionNumber: 'asc' },
    });
  }

  async findOne(ownerId: number, campaignId: number, id: number) {
    await this.assertCampaignOwnership(ownerId, campaignId);
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
   */
  async end(ownerId: number, campaignId: number, id: number) {
    const session = await this.findOne(ownerId, campaignId, id);
    if (session.status === 'planned') {
      throw new BadRequestException(
        `Session ${id} was never started; nothing to end`,
      );
    }
    if (session.status === 'ended') return session;
    return this.prisma.session.update({
      where: { id },
      data: { status: 'ended', endedAt: new Date() },
    });
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
