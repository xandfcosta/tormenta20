import { Module } from '@nestjs/common';
import { CampaignInvitesController } from './campaign-invites.controller';
import { CampaignsController } from './campaigns.controller';
import { CampaignsService } from './campaigns.service';

@Module({
  controllers: [CampaignsController, CampaignInvitesController],
  providers: [CampaignsService],
  exports: [CampaignsService],
})
export class CampaignsModule {}
