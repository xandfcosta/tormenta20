jest.mock('../prisma/prisma.service', () => ({
  PrismaService: class {},
}));

import { UnauthorizedException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Test } from '@nestjs/testing';
import type { Request } from 'express';
import { AuthService } from './auth.service';
import type { AuthUser } from './auth.service';
import { JwtStrategy, cookieExtractor } from './jwt.strategy';

/**
 * Unit specs for JwtStrategy. Constructor wires passport-jwt against a
 * cookie+bearer extractor; validate() resolves the AuthUser by id.
 * PrismaService is stubbed at module level so the generated client isn't
 * required at unit-test time.
 */
class FakeConfigService {
  private values: Record<string, string | undefined> = {};
  set(key: string, value: string | undefined): void {
    this.values[key] = value;
  }
  get<T>(key: string): T | undefined {
    return this.values[key] as T | undefined;
  }
}

class FakeAuthService {
  findById = jest.fn<Promise<AuthUser | null>, [number]>();
}

async function buildStrategy(opts: {
  jwtSecret?: string;
  cookieName?: string;
} = {}): Promise<{ strategy: JwtStrategy; auth: FakeAuthService }> {
  const config = new FakeConfigService();
  config.set('JWT_SECRET', opts.jwtSecret ?? 'test-secret');
  if (opts.cookieName !== undefined) {
    config.set('COOKIE_NAME', opts.cookieName);
  }
  const auth = new FakeAuthService();
  const moduleRef = await Test.createTestingModule({
    providers: [
      JwtStrategy,
      { provide: ConfigService, useValue: config },
      { provide: AuthService, useValue: auth },
    ],
  }).compile();
  return { strategy: moduleRef.get(JwtStrategy), auth };
}

describe('cookieExtractor', () => {
  function req(cookies?: Record<string, string>): Request {
    return { cookies } as unknown as Request;
  }

  it('returns the cookie value when the named cookie is present', () => {
    const extract = cookieExtractor('t20_session');
    expect(extract(req({ t20_session: 'abc.def.ghi' }))).toBe('abc.def.ghi');
  });

  it('returns null when the named cookie is missing', () => {
    const extract = cookieExtractor('t20_session');
    expect(extract(req({ other_cookie: 'x' }))).toBeNull();
  });

  it('returns null when the cookies object is absent', () => {
    const extract = cookieExtractor('t20_session');
    expect(extract(req(undefined))).toBeNull();
  });

  it('respects a custom cookie name', () => {
    const extract = cookieExtractor('custom');
    expect(extract(req({ custom: 'token-value' }))).toBe('token-value');
    expect(extract(req({ t20_session: 'wrong' }))).toBeNull();
  });
});

describe('JwtStrategy constructor', () => {
  it('throws a descriptive error when JWT_SECRET is not configured', async () => {
    const config = new FakeConfigService();
    const auth = new FakeAuthService();
    await expect(
      Test.createTestingModule({
        providers: [
          JwtStrategy,
          { provide: ConfigService, useValue: config },
          { provide: AuthService, useValue: auth },
        ],
      }).compile(),
    ).rejects.toThrow(/JWT_SECRET is required/);
  });

  it('boots cleanly when JWT_SECRET is set', async () => {
    const { strategy } = await buildStrategy({ jwtSecret: 's3cret' });
    expect(strategy).toBeInstanceOf(JwtStrategy);
  });
});

describe('JwtStrategy.validate', () => {
  it('returns the AuthUser resolved by AuthService.findById', async () => {
    const { strategy, auth } = await buildStrategy();
    const user: AuthUser = { id: 7, email: 'a@b.com', name: 'A' };
    auth.findById.mockResolvedValue(user);
    const result = await strategy.validate({ sub: 7, email: 'a@b.com' });
    expect(auth.findById).toHaveBeenCalledWith(7);
    expect(result).toBe(user);
  });

  it('throws UnauthorizedException when the user no longer exists', async () => {
    const { strategy, auth } = await buildStrategy();
    auth.findById.mockResolvedValue(null);
    await expect(
      strategy.validate({ sub: 999, email: 'gone@b.com' }),
    ).rejects.toBeInstanceOf(UnauthorizedException);
  });

  it('surfaces the standard "User no longer exists" message', async () => {
    const { strategy, auth } = await buildStrategy();
    auth.findById.mockResolvedValue(null);
    const err = await strategy
      .validate({ sub: 1, email: 'x@b.com' })
      .catch((e) => e as Error);
    expect(err.message).toMatch(/User no longer exists/);
  });
});
