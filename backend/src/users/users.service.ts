import { Injectable } from '@nestjs/common';
import { PrismaService } from '../prisma/prisma.service';

@Injectable()
export class UsersService {
  constructor(private readonly prisma: PrismaService) {}

  /**
   * Returns users the caller shares at least one campaign with, plus
   * the caller themselves. "Share a campaign" means either:
   *   (a) the caller is GM (Campaign.owner) and the other user owns a
   *       Character that is a member of that campaign, or
   *   (b) the caller owns a Character that is a member of a campaign
   *       whose GM is the other user.
   *
   * This is the only roster read — a global `listAll` was removed after
   * the whole-flow audit flagged it as an email/name enumeration surface.
   */
  async listVisibleTo(callerId: number) {
    const [asGm, asPlayer] = await Promise.all([
      this.prisma.campaignMember.findMany({
        where: { campaign: { ownerId: callerId } },
        select: { character: { select: { ownerId: true } } },
      }),
      this.prisma.campaignMember.findMany({
        where: { character: { ownerId: callerId } },
        select: { campaign: { select: { ownerId: true } } },
      }),
    ]);
    const visibleIds = new Set<number>([callerId]);
    for (const m of asGm) visibleIds.add(m.character.ownerId);
    for (const m of asPlayer) visibleIds.add(m.campaign.ownerId);
    return this.prisma.user.findMany({
      where: { id: { in: [...visibleIds] } },
      orderBy: { createdAt: 'desc' },
      select: { id: true, email: true, name: true, createdAt: true },
    });
  }
}
