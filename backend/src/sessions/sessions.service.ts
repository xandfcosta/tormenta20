import {
  BadRequestException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CampaignsService } from '../campaigns/campaigns.service';
import { CreateSessionDto, UpdateSessionDto } from './dto/session.dto';

export type SessionStatus = 'planned' | 'active' | 'ended';

@Injectable()
export class SessionsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly campaigns: CampaignsService,
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
   * Transition planned → active. Rejects if the session already ended
   * (that path is a fresh session, not a re-open), and idempotent when
   * already active — we return the current row without touching startedAt.
   */
  async start(ownerId: number, campaignId: number, id: number) {
    const session = await this.findOne(ownerId, campaignId, id);
    if (session.status === 'ended') {
      throw new BadRequestException(
        `Session ${id} has ended; create a new session to keep playing`,
      );
    }
    if (session.status === 'active') return session;
    return this.prisma.session.update({
      where: { id },
      data: { status: 'active', startedAt: new Date() },
    });
  }

  /**
   * Transition active → ended. Rejects on planned (nothing to end) and
   * idempotent on ended (returns the current row).
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
}
