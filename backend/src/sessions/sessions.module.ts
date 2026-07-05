import { Module } from '@nestjs/common';
import { CampaignsModule } from '../campaigns/campaigns.module';
import { SessionsController } from './sessions.controller';
import { SessionsService } from './sessions.service';

@Module({
  imports: [CampaignsModule],
  controllers: [SessionsController],
  providers: [SessionsService],
})
export class SessionsModule {}
