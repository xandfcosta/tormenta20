import {
  Body,
  Controller,
  Delete,
  Get,
  Param,
  ParseIntPipe,
  Patch,
  Post,
  UseGuards,
} from '@nestjs/common';
import { CurrentUser } from '../auth/current-user.decorator';
import type { AuthUser } from '../auth/auth-user.type';
import { JwtAuthGuard } from '../auth/jwt-auth.guard';
import { SessionsService } from './sessions.service';
import { CreateSessionDto, UpdateSessionDto } from './dto/session.dto';

@UseGuards(JwtAuthGuard)
@Controller('campaigns/:campaignId/sessions')
export class SessionsController {
  constructor(private readonly sessions: SessionsService) {}

  @Get()
  list(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
  ) {
    return this.sessions.listForCaller(user.id, campaignId);
  }

  @Get(':id')
  async findOne(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    const { session } = await this.sessions.findOneForCaller(
      user.id,
      campaignId,
      id,
    );
    return session;
  }

  @Post()
  create(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Body() dto: CreateSessionDto,
  ) {
    return this.sessions.create(user.id, campaignId, dto);
  }

  @Patch(':id')
  update(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateSessionDto,
  ) {
    return this.sessions.update(user.id, campaignId, id, dto);
  }

  @Delete(':id')
  remove(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.sessions.remove(user.id, campaignId, id);
  }

  @Post(':id/start')
  start(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.sessions.start(user.id, campaignId, id);
  }

  @Post(':id/end')
  end(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.sessions.end(user.id, campaignId, id);
  }

  @Post(':id/clear-tracker')
  clearTracker(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.sessions.clearTracker(user.id, campaignId, id);
  }
}
