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
   * Add a character to the campaign. Requires the character to exist —
   * v1 does NOT require the character's owner to grant access. That
   * looser check is deliberate for solo GMs prototyping party rosters;
   * a stricter invite flow (character-owner acceptance) can layer on
   * later without changing this signature.
   */
  async add(ownerId: number, campaignId: number, dto: AddMemberDto) {
    await this.assertCampaignOwnership(ownerId, campaignId);
    const character = await this.prisma.character.findUnique({
      where: { id: dto.characterId },
      select: { id: true },
    });
    if (!character) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Character ${dto.characterId} not found`,
        fieldErrors: { characterId: [`Character does not exist`] },
      });
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

  async remove(ownerId: number, campaignId: number, memberId: number) {
    await this.findOne(ownerId, campaignId, memberId);
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
