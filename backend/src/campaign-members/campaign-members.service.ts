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
   * Member-aware: any player in the campaign can view the party, not
   * just the GM — the roster is shared table state.
   */
  async list(userId: number, campaignId: number) {
    await this.campaigns.resolveAccess(userId, campaignId);
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
    // Join authorization (owner joins freely; everyone else needs a valid
    // invite token) + campaign existence are the campaigns context's
    // rules — delegate so the campaign row + token stay private to it.
    await this.campaigns.assertCanJoin(callerId, campaignId, dto.inviteToken);
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
    // One player character per user per campaign. A player brings a single
    // PC to a table; GM-role entries (recurring NPCs) are exempt so a mestre
    // can still run several. Enforced here rather than by a DB constraint
    // because the rule is per-owner, not per-character.
    if ((dto.role ?? 'player') === 'player') {
      const priorPc = await this.prisma.campaignMember.findFirst({
        where: {
          campaignId,
          role: 'player',
          character: { ownerId: callerId },
        },
        select: { id: true },
      });
      if (priorPc) {
        throw new ConflictException({
          statusCode: 409,
          error: 'Conflict',
          message: `You already have a character in campaign ${campaignId}`,
          fieldErrors: {
            characterId: ['Você já tem um personagem nesta campanha'],
          },
        });
      }
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

  /**
   * Player-controlled characters of a campaign with live combat stats —
   * for the realtime tracker's "add the whole party" action. Keeps the
   * campaignMember + character reads inside this context instead of the
   * WS gateway.
   */
  async listPlayerCombatants(campaignId: number): Promise<
    {
      characterId: number;
      name: string;
      hpCurrent: number;
      hpMax: number;
      mpCurrent: number;
      mpMax: number;
    }[]
  > {
    const members = await this.prisma.campaignMember.findMany({
      where: { campaignId, role: 'player' },
      select: {
        character: {
          select: {
            id: true,
            name: true,
            hpCurrent: true,
            hpMax: true,
            mpCurrent: true,
            mpMax: true,
          },
        },
      },
    });
    return members.map(({ character: c }) => ({
      characterId: c.id,
      name: c.name,
      hpCurrent: c.hpCurrent,
      hpMax: c.hpMax,
      mpCurrent: c.mpCurrent,
      mpMax: c.mpMax,
    }));
  }

  /** Character ids of every member of a campaign — for session-wide rest. */
  async listMemberCharacterIds(campaignId: number): Promise<number[]> {
    const members = await this.prisma.campaignMember.findMany({
      where: { campaignId },
      select: { characterId: true },
    });
    return members.map((m) => m.characterId);
  }

  /**
   * Resolve a character into a session combatant, validating that it is a
   * member of the campaign and that the caller may add it (owns it, or is
   * the campaign GM). Returns its combat stats. Encapsulates the
   * character + campaign + membership reads the WS gateway used to do
   * inline. Throws NotFound / BadRequest / Forbidden.
   */
  async resolveCombatant(
    callerId: number,
    campaignId: number,
    characterId: number,
  ): Promise<{
    name: string;
    hpCurrent: number;
    hpMax: number;
    mpCurrent: number;
    mpMax: number;
  }> {
    const [character, campaign, member] = await Promise.all([
      this.prisma.character.findUnique({
        where: { id: characterId },
        select: {
          name: true,
          ownerId: true,
          hpCurrent: true,
          hpMax: true,
          mpCurrent: true,
          mpMax: true,
        },
      }),
      this.prisma.campaign.findUnique({
        where: { id: campaignId },
        select: { ownerId: true },
      }),
      this.prisma.campaignMember.findUnique({
        where: { campaignId_characterId: { campaignId, characterId } },
        select: { id: true },
      }),
    ]);
    if (!character) {
      throw new NotFoundException(`Character ${characterId} not found`);
    }
    if (!campaign) {
      throw new NotFoundException(`Campaign ${campaignId} not found`);
    }
    if (!member) {
      throw new BadRequestException(
        `Character ${characterId} is not a member of campaign ${campaignId}`,
      );
    }
    if (callerId !== character.ownerId && callerId !== campaign.ownerId) {
      throw new ForbiddenException(
        `Caller ${callerId} is neither the GM of campaign ${campaignId} nor the owner of character ${characterId}`,
      );
    }
    return {
      name: character.name,
      hpCurrent: character.hpCurrent,
      hpMax: character.hpMax,
      mpCurrent: character.mpCurrent,
      mpMax: character.mpMax,
    };
  }
}
