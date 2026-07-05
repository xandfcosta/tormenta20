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
import type { AuthUser } from '../auth/auth.service';
import { JwtAuthGuard } from '../auth/jwt-auth.guard';
import { CampaignMembersService } from './campaign-members.service';
import {
  AddMemberDto,
  UpdateMemberDto,
} from './dto/campaign-member.dto';

@UseGuards(JwtAuthGuard)
@Controller('campaigns/:campaignId/members')
export class CampaignMembersController {
  constructor(private readonly members: CampaignMembersService) {}

  @Get()
  list(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
  ) {
    return this.members.list(user.id, campaignId);
  }

  @Post()
  add(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Body() dto: AddMemberDto,
  ) {
    return this.members.add(user.id, campaignId, dto);
  }

  @Patch(':id')
  updateRole(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
    @Body() dto: UpdateMemberDto,
  ) {
    return this.members.updateRole(user.id, campaignId, id, dto);
  }

  @Delete(':id')
  remove(
    @CurrentUser() user: AuthUser,
    @Param('campaignId', ParseIntPipe) campaignId: number,
    @Param('id', ParseIntPipe) id: number,
  ) {
    return this.members.remove(user.id, campaignId, id);
  }
}
