import { Module } from '@nestjs/common';
import { CampaignsModule } from '../campaigns/campaigns.module';
import { CampaignMembersController } from './campaign-members.controller';
import { CampaignMembersService } from './campaign-members.service';

@Module({
  imports: [CampaignsModule],
  controllers: [CampaignMembersController],
  providers: [CampaignMembersService],
  exports: [CampaignMembersService],
})
export class CampaignMembersModule {}
