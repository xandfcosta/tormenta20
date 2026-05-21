import {
  Body,
  Controller,
  Get,
  HttpCode,
  Post,
  Res,
  UseGuards,
} from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import type { CookieOptions, Response } from 'express';
import { AuthService } from './auth.service';
import type { AuthUser } from './auth.service';
import { CurrentUser } from './current-user.decorator';
import { LoginDto, RegisterDto } from './dto/auth.dto';
import { JwtAuthGuard } from './jwt-auth.guard';

const ONE_WEEK_MS = 7 * 24 * 60 * 60 * 1000;

@Controller('auth')
export class AuthController {
  constructor(
    private readonly auth: AuthService,
    private readonly config: ConfigService,
  ) {}

  private get cookieName(): string {
    return this.config.get<string>('COOKIE_NAME') ?? 't20_session';
  }

  private cookieOptions(): CookieOptions {
    const isProd = this.config.get<string>('NODE_ENV') === 'production';
    return {
      httpOnly: true,
      sameSite: 'lax',
      secure: isProd,
      path: '/',
      maxAge: ONE_WEEK_MS,
    };
  }

  @Post('register')
  async register(
    @Body() dto: RegisterDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<AuthUser> {
    const user = await this.auth.register(dto);
    const token = this.auth.signToken(user);
    res.cookie(this.cookieName, token, this.cookieOptions());
    return user;
  }

  @HttpCode(200)
  @Post('login')
  async login(
    @Body() dto: LoginDto,
    @Res({ passthrough: true }) res: Response,
  ): Promise<AuthUser> {
    const user = await this.auth.validate(dto.email, dto.password);
    const token = this.auth.signToken(user);
    res.cookie(this.cookieName, token, this.cookieOptions());
    return user;
  }

  @HttpCode(204)
  @Post('logout')
  logout(@Res({ passthrough: true }) res: Response): void {
    res.clearCookie(this.cookieName, { ...this.cookieOptions(), maxAge: 0 });
  }

  @UseGuards(JwtAuthGuard)
  @Get('me')
  me(@CurrentUser() user: AuthUser): AuthUser {
    return user;
  }
}
