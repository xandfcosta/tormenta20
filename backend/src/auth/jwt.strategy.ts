import { Injectable, UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { PassportStrategy } from '@nestjs/passport';
import type { Request } from 'express';
import { ExtractJwt, Strategy } from 'passport-jwt';
import { AuthService } from './auth.service';
import type { AuthUser, JwtPayload } from './auth.service';

/**
 * Returns the JWT extractor used by passport-jwt to pull the token out of
 * the session cookie. Exported so we can unit-test it directly without
 * spinning up the full passport pipeline.
 */
export function cookieExtractor(cookieName: string) {
  return (req: Request): string | null => {
    const cookies = (req as Request & { cookies?: Record<string, string> }).cookies;
    return cookies?.[cookieName] ?? null;
  };
}

@Injectable()
export class JwtStrategy extends PassportStrategy(Strategy) {
  constructor(
    config: ConfigService,
    private readonly auth: AuthService,
  ) {
    const secret = config.get<string>('JWT_SECRET');
    if (!secret) {
      throw new Error('JWT_SECRET is required (got: undefined). Set it in backend/.env');
    }
    const cookieName = config.get<string>('COOKIE_NAME') ?? 't20_session';
    super({
      jwtFromRequest: ExtractJwt.fromExtractors([
        cookieExtractor(cookieName),
        ExtractJwt.fromAuthHeaderAsBearerToken(),
      ]),
      ignoreExpiration: false,
      secretOrKey: secret,
    });
  }

  async validate(payload: JwtPayload): Promise<AuthUser> {
    const user = await this.auth.findById(payload.sub);
    if (!user) throw new UnauthorizedException('User no longer exists');
    return user;
  }
}
