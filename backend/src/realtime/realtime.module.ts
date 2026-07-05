import { Module } from '@nestjs/common';
import { AuthModule } from '../auth/auth.module';
import { SessionsModule } from '../sessions/sessions.module';
import { RealtimeGateway } from './realtime.gateway';
import { SessionStateService } from './session-state.service';

@Module({
  imports: [AuthModule, SessionsModule],
  providers: [RealtimeGateway, SessionStateService],
  exports: [RealtimeGateway, SessionStateService],
})
export class RealtimeModule {}
