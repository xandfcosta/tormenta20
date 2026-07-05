import { Controller, Get, UseGuards } from '@nestjs/common';
import { CurrentUser } from '../auth/current-user.decorator';
import type { AuthUser } from '../auth/auth.service';
import { JwtAuthGuard } from '../auth/jwt-auth.guard';
import { UsersService } from './users.service';

@UseGuards(JwtAuthGuard)
@Controller('users')
export class UsersController {
  constructor(private readonly users: UsersService) {}

  /**
   * Scoped roster: caller + users who share at least one campaign with
   * them. The pre-audit version returned every user in the system,
   * which leaked emails to any authenticated caller.
   */
  @Get()
  list(@CurrentUser() user: AuthUser) {
    return this.users.listVisibleTo(user.id);
  }
}
