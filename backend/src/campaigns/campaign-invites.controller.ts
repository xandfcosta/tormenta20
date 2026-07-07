import { Controller, Get, NotFoundException, Param } from '@nestjs/common';
import { CampaignsService } from './campaigns.service';

/**
 * Anonymous invite-token resolver. Deliberately does NOT hang off the
 * `/campaigns` prefix (which is guarded by JwtAuthGuard at the
 * controller level). A public route that returns only campaign id + name
 * — no members, no sessions — so a would-be joiner can preview the
 * mesa before logging in / registering.
 */
@Controller('invites')
export class CampaignInvitesController {
  constructor(private readonly campaigns: CampaignsService) {}

  @Get(':token')
  async resolve(@Param('token') token: string) {
    const found = await this.campaigns.resolveInviteToken(token);
    if (!found) {
      throw new NotFoundException(
        `Convite ${token} inválido ou expirado`,
      );
    }
    return found;
  }
}
