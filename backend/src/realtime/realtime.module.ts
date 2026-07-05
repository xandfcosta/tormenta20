import { forwardRef, Module } from '@nestjs/common';
import { AuthModule } from '../auth/auth.module';
import {
  SessionsModule,
  SESSION_STATE_SERVICE,
} from '../sessions/sessions.module';
import { RealtimeGateway } from './realtime.gateway';
import { SessionStateService } from './session-state.service';

@Module({
  imports: [AuthModule, forwardRef(() => SessionsModule)],
  providers: [
    RealtimeGateway,
    SessionStateService,
    /* Re-provide the same singleton under the string token that
     * SessionsService.state consumes — avoids a runtime import cycle
     * that would drag Prisma into unit tests. */
    { provide: SESSION_STATE_SERVICE, useExisting: SessionStateService },
  ],
  exports: [RealtimeGateway, SessionStateService],
})
export class RealtimeModule {}
