import { forwardRef, Module } from '@nestjs/common';
import { CampaignsModule } from '../campaigns/campaigns.module';
import { RealtimeModule } from '../realtime/realtime.module';
import { SessionsController } from './sessions.controller';
import { SessionsService } from './sessions.service';

/* Re-export the token so RealtimeModule can align its provider. */
export { SESSION_STATE_SERVICE } from './sessions.service';

@Module({
  /* forwardRef breaks the circular import: RealtimeModule already
   * pulls SessionsService for ownership checks, and SessionsService
   * now needs SessionStateService for clearTracker. Nest resolves the
   * cycle at boot. */
  imports: [CampaignsModule, forwardRef(() => RealtimeModule)],
  controllers: [SessionsController],
  providers: [SessionsService],
  exports: [SessionsService],
})
export class SessionsModule {}
