import { randomBytes } from 'node:crypto';
import {
  BadRequestException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CreateCampaignDto, UpdateCampaignDto } from './dto/campaign.dto';

/**
 * Invite tokens are 24 chars of base64url — 144 bits of entropy. Long
 * enough that guessing is not feasible; short enough to fit in a URL
 * comfortably (`/join/<token>`). Base64url is chosen over hex so the
 * URL stays copy-pasteable without percent-encoding.
 */
const INVITE_TOKEN_BYTES = 18;

function generateInviteToken(): string {
  return randomBytes(INVITE_TOKEN_BYTES).toString('base64url');
}

@Injectable()
export class CampaignsService {
  constructor(private readonly prisma: PrismaService) {}

  list(ownerId: number) {
    return this.prisma.campaign.findMany({
      where: { ownerId },
      orderBy: { updatedAt: 'desc' },
    });
  }

  async findOne(ownerId: number, id: number) {
    const campaign = await this.prisma.campaign.findUnique({ where: { id } });
    if (!campaign) throw new NotFoundException(`Campaign ${id} not found`);
    if (campaign.ownerId !== ownerId) {
      throw new ForbiddenException(`Campaign ${id} belongs to another user`);
    }
    return campaign;
  }

  create(ownerId: number, dto: CreateCampaignDto) {
    return this.prisma.campaign.create({
      data: {
        ownerId,
        name: dto.name.trim(),
        description: dto.description?.trim() ?? null,
      },
    });
  }

  async update(ownerId: number, id: number, dto: UpdateCampaignDto) {
    await this.findOne(ownerId, id);
    const data: { name?: string; description?: string | null } = {};
    if (dto.name !== undefined) data.name = dto.name.trim();
    if (dto.description !== undefined) {
      data.description = dto.description.trim() || null;
    }
    if (Object.keys(data).length === 0) {
      throw new BadRequestException('No fields to update');
    }
    return this.prisma.campaign.update({ where: { id }, data });
  }

  async remove(ownerId: number, id: number) {
    await this.findOne(ownerId, id);
    await this.prisma.campaign.delete({ where: { id } });
    return { id };
  }

  /**
   * Rotate the campaign's invite token. Only the GM (campaign owner)
   * can call this. Rotating replaces any previously-issued token, so
   * a link the GM shared and later regretted goes 404 immediately.
   */
  async rotateInviteToken(callerId: number, campaignId: number) {
    await this.findOne(callerId, campaignId);
    const token = generateInviteToken();
    const updated = await this.prisma.campaign.update({
      where: { id: campaignId },
      data: { inviteToken: token },
      select: { id: true, inviteToken: true },
    });
    return { campaignId: updated.id, token: updated.inviteToken as string };
  }

  /**
   * Resolve an invite token to its campaign (id + name). Called by the
   * anonymous invite landing page so a would-be joiner sees which mesa
   * the GM is inviting them into before they log in. Never throws —
   * returns null on unknown/rotated tokens so the frontend can show a
   * "convite expirado" message without a 500.
   */
  async resolveInviteToken(token: string) {
    if (!token) return null;
    const campaign = await this.prisma.campaign.findUnique({
      where: { inviteToken: token },
      select: { id: true, name: true },
    });
    if (!campaign) return null;
    return { campaignId: campaign.id, campaignName: campaign.name };
  }
}
