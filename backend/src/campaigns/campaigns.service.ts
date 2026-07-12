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

/** Caller's relationship to a campaign. `gm` = campaign owner; `player`
 * = owns a character that is a CampaignMember. Anyone else has no
 * access. There is no per-user role column — a user is the GM of the
 * campaigns they own and a player in the ones their characters joined. */
export type CampaignRole = 'gm' | 'player';

@Injectable()
export class CampaignsService {
  constructor(private readonly prisma: PrismaService) {}

  /**
   * Campaigns the user can see: ones they own (as GM) plus ones a
   * character of theirs has joined (as player). Each row is tagged with
   * the caller's `role` so the client can branch GM vs player views
   * without a second round-trip. Pre-membership this returned owned
   * campaigns only, which left players with an empty list.
   */
  async list(userId: number) {
    const campaigns = await this.prisma.campaign.findMany({
      where: {
        OR: [
          { ownerId: userId },
          { members: { some: { character: { ownerId: userId } } } },
        ],
      },
      orderBy: { updatedAt: 'desc' },
    });
    return campaigns.map((c) => ({
      ...c,
      role: (c.ownerId === userId ? 'gm' : 'player') as CampaignRole,
    }));
  }

  /**
   * Owner-only lookup. Kept as the guard for every **write** path
   * (update / delete / invite rotation / session lifecycle) — those
   * must stay GM-exclusive. Reads go through {@link resolveAccess}.
   */
  async findOne(ownerId: number, id: number) {
    const campaign = await this.prisma.campaign.findUnique({ where: { id } });
    if (!campaign) throw new NotFoundException(`Campaign ${id} not found`);
    if (campaign.ownerId !== ownerId) {
      throw new ForbiddenException(`Campaign ${id} belongs to another user`);
    }
    return campaign;
  }

  /**
   * Member-aware access for **read** paths. Returns the campaign plus
   * the caller's role, or throws NotFound (missing) / Forbidden (caller
   * is neither owner nor a member via one of their characters).
   */
  async resolveAccess(userId: number, id: number) {
    const campaign = await this.prisma.campaign.findUnique({ where: { id } });
    if (!campaign) throw new NotFoundException(`Campaign ${id} not found`);
    if (campaign.ownerId === userId) return { campaign, role: 'gm' as CampaignRole };
    const membership = await this.prisma.campaignMember.findFirst({
      where: { campaignId: id, character: { ownerId: userId } },
      select: { id: true },
    });
    if (membership) return { campaign, role: 'player' as CampaignRole };
    throw new ForbiddenException(`Campaign ${id} is not accessible`);
  }

  /**
   * Join authorization for adding a character to a campaign. The owner
   * (GM) may add their own characters/NPCs freely; everyone else must
   * present the campaign's CURRENT invite token. Throws NotFound if the
   * campaign is gone, Forbidden if a non-owner has no valid token. Keeps
   * the campaign row + token rule private to this context (called by
   * CampaignMembersService.add).
   */
  async assertCanJoin(
    userId: number,
    campaignId: number,
    inviteToken?: string,
  ): Promise<void> {
    const campaign = await this.prisma.campaign.findUnique({
      where: { id: campaignId },
      select: { id: true, ownerId: true, inviteToken: true },
    });
    if (!campaign) {
      throw new NotFoundException(`Campaign ${campaignId} not found`);
    }
    if (campaign.ownerId === userId) return;
    if (campaign.inviteToken === null || inviteToken !== campaign.inviteToken) {
      throw new ForbiddenException(
        `A valid invite token is required to join campaign ${campaignId}`,
      );
    }
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
