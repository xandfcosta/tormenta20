import {
  BadRequestException,
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';
import { CampaignsService } from '../campaigns/campaigns.service';
import {
  AddMemberDto,
  UpdateMemberDto,
} from './dto/campaign-member.dto';

@Injectable()
export class CampaignMembersService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly campaigns: CampaignsService,
  ) {}

  private async assertCampaignOwnership(ownerId: number, campaignId: number) {
    await this.campaigns.findOne(ownerId, campaignId);
  }

  /**
   * List members of a campaign. Joins the Character row so callers can
   * render a roster (name / level / class) without a follow-up call.
   */
  async list(ownerId: number, campaignId: number) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    return this.prisma.campaignMember.findMany({
      where: { campaignId },
      orderBy: { addedAt: 'asc' },
      include: {
        character: {
          select: {
            id: true,
            name: true,
            level: true,
            classes: { select: { className: true, level: true } },
          },
        },
      },
    });
  }

  /**
   * Add a character to a campaign. **The caller must own the character
   * being added.** This closes OC1 from the whole-flow audit: the
   * pre-fix rule ("caller must be GM of campaign") let any GM slot any
   * user's PC into their own table without consent from the player.
   *
   * New semantics:
   *   - Player self-joins a campaign: caller owns the character being
   *     added; campaign must exist. No consent step needed because
   *     the actor is the character owner.
   *   - GM adds their own character (PC/NPC) to a campaign they run:
   *     works because caller is the character owner.
   *   - GM adds someone else's character: rejected with 403.
   *
   * A full invite/accept flow (`Invitation` model) would restore the
   * "GM invites → player accepts" pattern; deliberately out of scope
   * for the MVP fix — a GM can always ask the player to self-add.
   */
  async add(callerId: number, campaignId: number, dto: AddMemberDto) {
    const campaign = await this.prisma.campaign.findUnique({
      where: { id: campaignId },
      select: { id: true, inviteToken: true },
    });
    if (!campaign) {
      throw new NotFoundException(`Campaign ${campaignId} not found`);
    }
    // Invite token flow: the token must match the campaign's current
    // token. Rotating the token invalidates any older link (this is
    // exactly why rotation exists — a leaked/regretted link goes 404
    // immediately once the GM rotates).
    if (dto.inviteToken !== undefined) {
      if (
        campaign.inviteToken === null ||
        campaign.inviteToken !== dto.inviteToken
      ) {
        throw new ForbiddenException(
          `Invite token invalid or expired for campaign ${campaignId}`,
        );
      }
    }
    const character = await this.prisma.character.findUnique({
      where: { id: dto.characterId },
      select: { id: true, ownerId: true },
    });
    if (!character) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Character ${dto.characterId} not found`,
        fieldErrors: { characterId: [`Character does not exist`] },
      });
    }
    if (character.ownerId !== callerId) {
      throw new ForbiddenException(
        `Cannot add a character you don't own (character ${dto.characterId})`,
      );
    }
    const existing = await this.prisma.campaignMember.findUnique({
      where: {
        campaignId_characterId: {
          campaignId,
          characterId: dto.characterId,
        },
      },
      select: { id: true },
    });
    if (existing) {
      throw new ConflictException({
        statusCode: 409,
        error: 'Conflict',
        message: `Character ${dto.characterId} already in campaign ${campaignId}`,
        fieldErrors: { characterId: [`Already a member`] },
      });
    }
    return this.prisma.campaignMember.create({
      data: {
        campaignId,
        characterId: dto.characterId,
        role: dto.role ?? 'player',
      },
    });
  }

  async findOne(ownerId: number, campaignId: number, memberId: number) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    const member = await this.prisma.campaignMember.findUnique({
      where: { id: memberId },
    });
    if (!member || member.campaignId !== campaignId) {
      throw new NotFoundException(`Member ${memberId} not found`);
    }
    return member;
  }

  async updateRole(
    ownerId: number,
    campaignId: number,
    memberId: number,
    dto: UpdateMemberDto,
  ) {
    await this.findOne(ownerId, campaignId, memberId);
    return this.prisma.campaignMember.update({
      where: { id: memberId },
      data: { role: dto.role },
    });
  }

  /**
   * Remove a member from a campaign. Either the campaign owner (GM
   * kicking a player, or removing an NPC) **or** the character owner
   * (player leaving the campaign) can drive the deletion. This closes
   * OI1 — pre-fix the GM was the only allowed actor, so players had no
   * way to leave a campaign, mirror-image of the OC1 consent leak.
   */
  async remove(callerId: number, campaignId: number, memberId: number) {
    const member = await this.prisma.campaignMember.findUnique({
      where: { id: memberId },
      select: {
        id: true,
        campaignId: true,
        campaign: { select: { ownerId: true } },
        character: { select: { ownerId: true } },
      },
    });
    if (!member || member.campaignId !== campaignId) {
      throw new NotFoundException(`Member ${memberId} not found`);
    }
    const isGm = member.campaign.ownerId === callerId;
    const isCharacterOwner = member.character.ownerId === callerId;
    if (!isGm && !isCharacterOwner) {
      throw new ForbiddenException(
        `You are neither the GM of this campaign nor the character's owner`,
      );
    }
    await this.prisma.campaignMember.delete({ where: { id: memberId } });
    return { id: memberId };
  }

  /**
   * Reverse lookup — list every campaign a character participates in.
   * The character must belong to `ownerId` (same rule as CharactersService).
   * Returns the join row + nested campaign so the UI can render both the
   * role (player/gm) and the campaign card in one hop.
   */
  async listForCharacter(ownerId: number, characterId: number) {
    const character = await this.prisma.character.findUnique({
      where: { id: characterId },
      select: { id: true, ownerId: true },
    });
    if (!character) {
      throw new NotFoundException(`Character ${characterId} not found`);
    }
    if (character.ownerId !== ownerId) {
      throw new ForbiddenException(
        `Character ${characterId} belongs to another user`,
      );
    }
    return this.prisma.campaignMember.findMany({
      where: { characterId },
      orderBy: { addedAt: 'asc' },
      include: {
        campaign: {
          select: {
            id: true,
            name: true,
            description: true,
            updatedAt: true,
          },
        },
      },
    });
  }
}
